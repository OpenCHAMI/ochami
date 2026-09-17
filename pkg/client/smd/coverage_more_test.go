// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package smd

import (
	"context"
	"net/http"
	"testing"

	"github.com/openchami/schemas/schemas/csm"
)

// TestAdditionalSMDClientPaths exercises the remaining successful SMD wrapper paths.
func TestAdditionalSMDClientPaths(t *testing.T) {
	tests := []struct {
		name       string
		call       func(*SMDClient) error
		wantMethod string
		wantPath   string
	}{
		{"component by NID", func(c *SMDClient) error { _, err := c.GetComponentsNid(context.Background(), 42, "tok"); return err }, http.MethodGet, "/State/Components/ByNID/42"},
		{"component endpoint", func(c *SMDClient) error { return c.GetComponentEndpoints(context.Background(), "tok", "x0")[0].Err }, http.MethodGet, "/Inventory/ComponentEndpoints/x0"},
		{"put redfish endpoint", func(c *SMDClient) error {
			return c.PutRedfishEndpoints(context.Background(), RedfishEndpointSlice{RedfishEndpoints: []csm.RedfishEndpoint{{ID: "x0"}}}, "tok")[0].Err
		}, http.MethodPut, "/Inventory/RedfishEndpoints/x0"},
		{"put redfish endpoint v2", func(c *SMDClient) error {
			return c.PutRedfishEndpointsV2(context.Background(), RedfishEndpointSliceV2{RedfishEndpoints: []RedfishEndpointV2{{RedfishEndpoint: csm.RedfishEndpoint{ID: "x0"}}}}, "tok")[0].Err
		}, http.MethodPut, "/Inventory/RedfishEndpoints/x0"},
		{"patch interface", func(c *SMDClient) error {
			return c.PatchEthernetInterfaces(context.Background(), []EthernetInterface{{ID: "eth0"}}, "tok")[0].Err
		}, http.MethodPatch, "/Inventory/EthernetInterfaces/eth0"},
		{"patch group", func(c *SMDClient) error {
			return c.PatchGroups(context.Background(), []Group{{Label: "compute"}}, "tok")[0].Err
		}, http.MethodPatch, "/groups/compute"},
		{"delete all component endpoints", func(c *SMDClient) error {
			_, err := c.DeleteComponentEndpointsAll(context.Background(), "tok")
			return err
		}, http.MethodDelete, "/Inventory/ComponentEndpoints"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotPath, gotAuth string
			c, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath, gotAuth = r.Method, r.URL.Path, r.Header.Get("Authorization")
				w.WriteHeader(http.StatusOK)
			})
			defer srv.Close()
			if err := tc.call(c); err != nil {
				t.Fatalf("call: %v", err)
			}
			if gotMethod != tc.wantMethod || gotPath != tc.wantPath {
				t.Errorf("request = %s %s, want %s %s", gotMethod, gotPath, tc.wantMethod, tc.wantPath)
			}
			if gotAuth != "Bearer tok" {
				t.Errorf("Authorization = %q", gotAuth)
			}
		})
	}
}

// TestSMDClientValidationBranches verifies invalid SMD inputs are rejected before requests are sent.
func TestSMDClientValidationBranches(t *testing.T) {
	c, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	defer srv.Close()

	results := c.PutRedfishEndpoints(context.Background(), RedfishEndpointSlice{RedfishEndpoints: []csm.RedfishEndpoint{{}}}, "")
	if len(results) != 1 || results[0].Err == nil {
		t.Fatalf("blank RFE results = %v", results)
	}
	results = c.PutRedfishEndpointsV2(context.Background(), RedfishEndpointSliceV2{RedfishEndpoints: []RedfishEndpointV2{{}}}, "")
	if len(results) != 1 || results[0].Err == nil {
		t.Fatalf("blank RFE v2 results = %v", results)
	}
	results = c.PatchEthernetInterfaces(context.Background(), []EthernetInterface{{}}, "")
	if len(results) != 1 || results[0].Err == nil {
		t.Fatalf("blank interface results = %v", results)
	}
	results = c.PatchGroups(context.Background(), []Group{{}}, "")
	if len(results) != 1 || results[0].Err == nil {
		t.Fatalf("blank group results = %v", results)
	}
}
