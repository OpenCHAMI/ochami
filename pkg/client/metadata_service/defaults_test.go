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

func TestAddDefaultsSpecs_OmitsLabels(t *testing.T) {
	var gotBody map[string]interface{}
	var gotPath, gotMethod string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.ClusterDefaults{})
	})
	defer srv.Close()

	defaults := []ClusterDefaultsSpec{
		{
			Name:                "demo-cluster-defaults",
			ClusterDefaultsSpec: api.ClusterDefaultsSpec{BaseURL: "https://demo.openchami.cluster:8443/cloud-init", ClusterName: "demo"},
		},
	}

	_, errs, err := c.AddDefaultsSpecs("", defaults)
	if err != nil {
		t.Fatalf("AddDefaultsSpecs func error: %v", err)
	}
	for _, e := range errs {
		if e != nil {
			t.Fatalf("AddDefaultsSpecs per-request error: %v", e)
		}
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/clusterdefaultss" {
		t.Errorf("path = %q, want /clusterdefaultss", gotPath)
	}
	if _, ok := gotBody["labels"]; ok {
		t.Errorf("simple request unexpectedly included labels: %+v", gotBody["labels"])
	}
	meta, _ := gotBody["metadata"].(map[string]interface{})
	if meta == nil || meta["name"] != "demo-cluster-defaults" {
		t.Errorf("metadata.name = %+v, want demo-cluster-defaults", meta)
	}
}

func TestAddDefaults_EnvelopeIncludesLabels(t *testing.T) {
	var gotBody map[string]interface{}
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.ClusterDefaults{})
	})
	defer srv.Close()

	defaults := []metadata_service_client.CreateClusterDefaultsRequest{
		{
			Metadata: fabrica.Metadata{Name: "demo-cluster-defaults", Labels: map[string]string{"env": "prod"}},
			Spec:     api.ClusterDefaultsSpec{BaseURL: "https://demo.openchami.cluster:8443/cloud-init", ClusterName: "demo"},
			Labels:   map[string]string{"env": "prod"},
		},
	}

	_, _, err := c.AddDefaults("", defaults)
	if err != nil {
		t.Fatalf("AddDefaults func error: %v", err)
	}

	labels, ok := gotBody["labels"].(map[string]interface{})
	if !ok || labels["env"] != "prod" {
		t.Errorf("envelope request labels = %+v, want env=prod", gotBody["labels"])
	}
}

func TestSetDefaultsSpec_UsesUIDEndpoint(t *testing.T) {
	var gotPath, gotMethod string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.ClusterDefaults{})
	})
	defer srv.Close()

	spec := api.ClusterDefaultsSpec{BaseURL: "https://demo.openchami.cluster:8443/cloud-init", ClusterName: "demo"}

	_, err := c.SetDefaultsSpec("", "clusterdefaults-abc", spec)
	if err != nil {
		t.Fatalf("SetDefaultsSpec error: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotPath != "/clusterdefaultss/clusterdefaults-abc" {
		t.Errorf("path = %q, want /clusterdefaultss/clusterdefaults-abc", gotPath)
	}
}
