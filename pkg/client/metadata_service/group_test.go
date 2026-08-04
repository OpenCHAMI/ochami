// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openchami/fabrica/pkg/fabrica"
	api "github.com/openchami/metadata-service/apis/cloud-init.openchami.io/v1"
	metadata_service_client "github.com/openchami/metadata-service/pkg/client"
	"github.com/rs/zerolog"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) (*MetadataServiceClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	c, err := NewClient(srv.URL, false, 5*time.Second, "", zerolog.New(io.Discard))
	if err != nil {
		srv.Close()
		t.Fatalf("failed to create client: %v", err)
	}
	return c, srv
}

func TestAddGroupSpecs_OmitsLabels(t *testing.T) {
	var gotBody map[string]interface{}
	var gotPath, gotMethod string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.Group{})
	})
	defer srv.Close()

	groups := []GroupSpec{
		{
			Name:      "compute-group",
			GroupSpec: api.GroupSpec{},
		},
	}

	_, errs, err := c.AddGroupSpecs("", groups)
	if err != nil {
		t.Fatalf("AddGroupSpecs func error: %v", err)
	}
	for _, e := range errs {
		if e != nil {
			t.Fatalf("AddGroupSpecs per-request error: %v", e)
		}
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/groups" {
		t.Errorf("path = %q, want /groups", gotPath)
	}
	if _, ok := gotBody["labels"]; ok {
		t.Errorf("simple request unexpectedly included labels: %+v", gotBody["labels"])
	}
	meta, _ := gotBody["metadata"].(map[string]interface{})
	if meta == nil || meta["name"] != "compute-group" {
		t.Errorf("metadata.name = %+v, want compute-group", meta)
	}
}

func TestAddGroups_EnvelopeIncludesLabels(t *testing.T) {
	var gotBody map[string]interface{}
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.Group{})
	})
	defer srv.Close()

	groups := []metadata_service_client.CreateGroupRequest{
		{
			Metadata: fabrica.Metadata{Name: "compute-group", Labels: map[string]string{"role": "compute"}},
			Spec:     api.GroupSpec{},
			Labels:   map[string]string{"role": "compute"},
		},
	}

	_, _, err := c.AddGroups("", groups)
	if err != nil {
		t.Fatalf("AddGroups func error: %v", err)
	}

	labels, ok := gotBody["labels"].(map[string]interface{})
	if !ok || labels["role"] != "compute" {
		t.Errorf("envelope request labels = %+v, want role=compute", gotBody["labels"])
	}
}

func TestSetGroupSpec_UsesUIDEndpoint(t *testing.T) {
	var gotPath, gotMethod string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.Group{})
	})
	defer srv.Close()

	spec := api.GroupSpec{}

	_, err := c.SetGroupSpec("", "grp-abc", spec)
	if err != nil {
		t.Fatalf("SetGroupSpec error: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotPath != "/groups/grp-abc" {
		t.Errorf("path = %q, want /groups/grp-abc", gotPath)
	}
}
