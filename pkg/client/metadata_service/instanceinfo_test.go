// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

import (
	"context"
	"net/http"
	"testing"

	"github.com/openchami/fabrica/pkg/fabrica"
	api "github.com/openchami/metadata-service/apis/cloud-init.openchami.io/v1"
	metadata_service_client "github.com/openchami/metadata-service/pkg/client"
)

// TestAddInstanceInfoSpecs_OmitsLabels verifies the simple API omits envelope labels.
func TestAddInstanceInfoSpecs_OmitsLabels(t *testing.T) {
	var gotBody map[string]interface{}
	var gotPath, gotMethod string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		decodeMetadataTestJSON(t, r, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		encodeMetadataTestJSON(t, w, api.InstanceInfo{})
	})
	defer srv.Close()

	instances := []InstanceInfoSpec{
		{
			Name:             "x1000c0s0b0n0-instance",
			InstanceInfoSpec: api.InstanceInfoSpec{InstanceID: "x1000c0s0b0n0"},
		},
	}

	results := c.AddInstanceInfoSpecs(context.Background(), "", instances)
	for _, e := range results.Errors() {
		if e != nil {
			t.Fatalf("AddInstanceInfoSpecs per-request error: %v", e)
		}
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/instanceinfos" {
		t.Errorf("path = %q, want /instanceinfos", gotPath)
	}
	if _, ok := gotBody["labels"]; ok {
		t.Errorf("simple request unexpectedly included labels: %+v", gotBody["labels"])
	}
	meta, _ := gotBody["metadata"].(map[string]interface{})
	if meta == nil || meta["name"] != "x1000c0s0b0n0-instance" {
		t.Errorf("metadata.name = %+v, want x1000c0s0b0n0-instance", meta)
	}
}

// TestAddInstanceInfos_EnvelopeIncludesLabels verifies the advanced API preserves resource labels.
func TestAddInstanceInfos_EnvelopeIncludesLabels(t *testing.T) {
	var gotBody map[string]interface{}
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		decodeMetadataTestJSON(t, r, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		encodeMetadataTestJSON(t, w, api.InstanceInfo{})
	})
	defer srv.Close()

	instances := []metadata_service_client.CreateInstanceInfoRequest{
		{
			Metadata: fabrica.Metadata{Name: "x1000c0s0b0n0-instance", Labels: map[string]string{"env": "prod"}},
			Spec:     api.InstanceInfoSpec{InstanceID: "x1000c0s0b0n0"},
			Labels:   map[string]string{"env": "prod"},
		},
	}

	if results := c.AddInstanceInfos(context.Background(), "", instances); results.HasErrors() {
		t.Fatalf("AddInstanceInfos errors: %v", results.Errors())
	}

	labels, ok := gotBody["labels"].(map[string]interface{})
	if !ok || labels["env"] != "prod" {
		t.Errorf("envelope request labels = %+v, want env=prod", gotBody["labels"])
	}
}

// TestSetInstanceInfoSpec_UsesUIDEndpoint verifies simple updates target the requested resource UID.
func TestSetInstanceInfoSpec_UsesUIDEndpoint(t *testing.T) {
	var gotPath, gotMethod string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		encodeMetadataTestJSON(t, w, api.InstanceInfo{})
	})
	defer srv.Close()

	spec := api.InstanceInfoSpec{InstanceID: "x1000c0s0b0n0"}

	_, err := c.SetInstanceInfoSpec(context.Background(), "", "instanceinfo-abc", spec)
	if err != nil {
		t.Fatalf("SetInstanceInfoSpec error: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotPath != "/instanceinfos/instanceinfo-abc" {
		t.Errorf("path = %q, want /instanceinfos/instanceinfo-abc", gotPath)
	}
}
