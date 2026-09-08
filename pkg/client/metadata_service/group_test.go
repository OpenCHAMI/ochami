// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

import (
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
	c, err := NewClient(srv.URL, 5*time.Second, "", zerolog.New(io.Discard))
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
		decodeMetadataTestJSON(t, r, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		encodeMetadataTestJSON(t, w, api.Group{})
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
		decodeMetadataTestJSON(t, r, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		encodeMetadataTestJSON(t, w, api.Group{})
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
		encodeMetadataTestJSON(t, w, api.Group{})
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

func TestEnvelopeSetMethods(t *testing.T) {
	tests := []struct {
		name     string
		wantPath string
		call     func(*MetadataServiceClient) error
	}{
		{"defaults", "/clusterdefaultss/uid", func(c *MetadataServiceClient) error {
			_, err := c.SetDefaults("tok", "uid", metadata_service_client.UpdateClusterDefaultsRequest{})
			return err
		}},
		{"group", "/groups/uid", func(c *MetadataServiceClient) error {
			_, err := c.SetGroup("tok", "uid", metadata_service_client.UpdateGroupRequest{})
			return err
		}},
		{"instance", "/instanceinfos/uid", func(c *MetadataServiceClient) error {
			_, err := c.SetInstanceInfo("tok", "uid", metadata_service_client.UpdateInstanceInfoRequest{})
			return err
		}},
		{"peer", "/wireguardpeers/uid", func(c *MetadataServiceClient) error {
			_, err := c.SetWireGuardPeer("tok", "uid", metadata_service_client.UpdateWireGuardPeerRequest{})
			return err
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotPath, gotAuth string
			c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath, gotAuth = r.Method, r.URL.Path, r.Header.Get("Authorization")
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{}`) //nolint:errcheck // test response writes are observed by the client
			})
			defer srv.Close()
			if err := tc.call(c); err != nil {
				t.Fatalf("set: %v", err)
			}
			if gotMethod != http.MethodPut || gotPath != tc.wantPath || gotAuth != "Bearer tok" {
				t.Errorf("request = %s %s auth=%q, want PUT %s", gotMethod, gotPath, gotAuth, tc.wantPath)
			}
		})
	}
}
