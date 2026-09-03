// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package smd

import (
	"net/http"
	"testing"

	"github.com/openchami/schemas/schemas/csm"
)

func TestAdditionalSMDClientPaths(t *testing.T) {
	tests := []struct {
		name       string
		call       func(*SMDClient) error
		wantMethod string
		wantPath   string
	}{
		{"component by NID", func(c *SMDClient) error { _, err := c.GetComponentsNid(42, "tok"); return err }, http.MethodGet, "/State/Components/ByNID/42"},
		{"component endpoint", func(c *SMDClient) error { _, _, err := c.GetComponentEndpoints("tok", "x0"); return err }, http.MethodGet, "/Inventory/ComponentEndpoints/x0"},
		{"put redfish endpoint", func(c *SMDClient) error {
			_, errs, err := c.PutRedfishEndpoints(RedfishEndpointSlice{RedfishEndpoints: []csm.RedfishEndpoint{{ID: "x0"}}}, "tok")
			if err == nil && len(errs) > 0 {
				err = errs[0]
			}
			return err
		}, http.MethodPut, "/Inventory/RedfishEndpoints/x0"},
		{"put redfish endpoint v2", func(c *SMDClient) error {
			_, errs, err := c.PutRedfishEndpointsV2(RedfishEndpointSliceV2{RedfishEndpoints: []RedfishEndpointV2{{RedfishEndpoint: csm.RedfishEndpoint{ID: "x0"}}}}, "tok")
			if err == nil && len(errs) > 0 {
				err = errs[0]
			}
			return err
		}, http.MethodPut, "/Inventory/RedfishEndpoints/x0"},
		{"patch interface", func(c *SMDClient) error {
			_, errs, err := c.PatchEthernetInterfaces([]EthernetInterface{{ID: "eth0"}}, "tok")
			if err == nil && len(errs) > 0 {
				err = errs[0]
			}
			return err
		}, http.MethodPatch, "/Inventory/EthernetInterfaces/eth0"},
		{"patch group", func(c *SMDClient) error {
			_, errs, err := c.PatchGroups([]Group{{Label: "compute"}}, "tok")
			if err == nil && len(errs) > 0 {
				err = errs[0]
			}
			return err
		}, http.MethodPatch, "/groups/compute"},
		{"delete all component endpoints", func(c *SMDClient) error { _, err := c.DeleteComponentEndpointsAll("tok"); return err }, http.MethodDelete, "/Inventory/ComponentEndpoints"},
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

func TestSMDClientValidationBranches(t *testing.T) {
	c, srv := newTestSMD(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	defer srv.Close()

	_, errs, err := c.PutRedfishEndpoints(RedfishEndpointSlice{RedfishEndpoints: []csm.RedfishEndpoint{{}}}, "")
	if err != nil || len(errs) != 1 || errs[0] == nil {
		t.Fatalf("blank RFE errors = %v, %v", errs, err)
	}
	_, errs, err = c.PutRedfishEndpointsV2(RedfishEndpointSliceV2{RedfishEndpoints: []RedfishEndpointV2{{}}}, "")
	if err != nil || len(errs) != 1 || errs[0] == nil {
		t.Fatalf("blank RFE v2 errors = %v, %v", errs, err)
	}
	_, errs, err = c.PatchEthernetInterfaces([]EthernetInterface{{}}, "")
	if err != nil || len(errs) != 1 || errs[0] == nil {
		t.Fatalf("blank interface errors = %v, %v", errs, err)
	}
	_, errs, err = c.PatchGroups([]Group{{}}, "")
	if err != nil || len(errs) != 1 || errs[0] == nil {
		t.Fatalf("blank group errors = %v, %v", errs, err)
	}
}
