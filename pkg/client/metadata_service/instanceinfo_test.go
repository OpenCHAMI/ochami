// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

import (
	"encoding/json"
	"net/http"
	"testing"

	api "github.com/OpenCHAMI/metadata-service/apis/cloud-init.openchami.io/v1"
	metadata_service_client "github.com/OpenCHAMI/metadata-service/pkg/client"
	"github.com/openchami/fabrica/pkg/fabrica"
)

func TestAddInstanceInfoSpecs_OmitsLabels(t *testing.T) {
	var gotBody map[string]interface{}
	var gotPath, gotMethod string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.InstanceInfo{})
	})
	defer srv.Close()

	instances := []InstanceInfoSpec{
		{
			Name:             "x1000c0s0b0n0-instance",
			InstanceInfoSpec: api.InstanceInfoSpec{InstanceID: "x1000c0s0b0n0"},
		},
	}

	_, errs, err := c.AddInstanceInfoSpecs("", instances)
	if err != nil {
		t.Fatalf("AddInstanceInfoSpecs func error: %v", err)
	}
	for _, e := range errs {
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

func TestAddInstanceInfos_EnvelopeIncludesLabels(t *testing.T) {
	var gotBody map[string]interface{}
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.InstanceInfo{})
	})
	defer srv.Close()

	instances := []metadata_service_client.CreateInstanceInfoRequest{
		{
			Metadata: fabrica.Metadata{Name: "x1000c0s0b0n0-instance", Labels: map[string]string{"env": "prod"}},
			Spec:     api.InstanceInfoSpec{InstanceID: "x1000c0s0b0n0"},
			Labels:   map[string]string{"env": "prod"},
		},
	}

	_, _, err := c.AddInstanceInfos("", instances)
	if err != nil {
		t.Fatalf("AddInstanceInfos func error: %v", err)
	}

	labels, ok := gotBody["labels"].(map[string]interface{})
	if !ok || labels["env"] != "prod" {
		t.Errorf("envelope request labels = %+v, want env=prod", gotBody["labels"])
	}
}

func TestSetInstanceInfoSpec_UsesUIDEndpoint(t *testing.T) {
	var gotPath, gotMethod string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.InstanceInfo{})
	})
	defer srv.Close()

	spec := api.InstanceInfoSpec{InstanceID: "x1000c0s0b0n0"}

	_, err := c.SetInstanceInfoSpec("", "instanceinfo-abc", spec)
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
