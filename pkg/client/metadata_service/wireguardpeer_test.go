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

// TestAddWireGuardPeerSpecs_OmitsLabels verifies the simple API omits envelope labels.
func TestAddWireGuardPeerSpecs_OmitsLabels(t *testing.T) {
	var gotBody map[string]interface{}
	var gotPath, gotMethod string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		decodeMetadataTestJSON(t, r, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		encodeMetadataTestJSON(t, w, api.WireGuardPeer{})
	})
	defer srv.Close()

	peers := []WireGuardPeerSpec{
		{
			Name:              "peer-nid001000",
			WireGuardPeerSpec: api.WireGuardPeerSpec{PublicKey: "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=", AllowedIP: "10.42.1.1/32"},
		},
	}

	results := c.AddWireGuardPeerSpecs(context.Background(), "", peers)
	for _, e := range results.Errors() {
		if e != nil {
			t.Fatalf("AddWireGuardPeerSpecs per-request error: %v", e)
		}
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/wireguardpeers" {
		t.Errorf("path = %q, want /wireguardpeers", gotPath)
	}
	if _, ok := gotBody["labels"]; ok {
		t.Errorf("simple request unexpectedly included labels: %+v", gotBody["labels"])
	}
	meta, _ := gotBody["metadata"].(map[string]interface{})
	if meta == nil || meta["name"] != "peer-nid001000" {
		t.Errorf("metadata.name = %+v, want peer-nid001000", meta)
	}
}

// TestAddWireGuardPeers_EnvelopeIncludesLabels verifies the advanced API preserves resource labels.
func TestAddWireGuardPeers_EnvelopeIncludesLabels(t *testing.T) {
	var gotBody map[string]interface{}
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		decodeMetadataTestJSON(t, r, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		encodeMetadataTestJSON(t, w, api.WireGuardPeer{})
	})
	defer srv.Close()

	peers := []metadata_service_client.CreateWireGuardPeerRequest{
		{
			Metadata: fabrica.Metadata{Name: "peer-nid001000", Labels: map[string]string{"env": "prod"}},
			Spec:     api.WireGuardPeerSpec{PublicKey: "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=", AllowedIP: "10.42.1.1/32"},
			Labels:   map[string]string{"env": "prod"},
		},
	}

	if results := c.AddWireGuardPeers(context.Background(), "", peers); results.HasErrors() {
		t.Fatalf("AddWireGuardPeers errors: %v", results.Errors())
	}

	labels, ok := gotBody["labels"].(map[string]interface{})
	if !ok || labels["env"] != "prod" {
		t.Errorf("envelope request labels = %+v, want env=prod", gotBody["labels"])
	}
}

// TestSetWireGuardPeerSpec_UsesUIDEndpoint verifies simple updates target the requested resource UID.
func TestSetWireGuardPeerSpec_UsesUIDEndpoint(t *testing.T) {
	var gotPath, gotMethod string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		encodeMetadataTestJSON(t, w, api.WireGuardPeer{})
	})
	defer srv.Close()

	spec := api.WireGuardPeerSpec{PublicKey: "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=", AllowedIP: "10.42.1.1/32"}

	_, err := c.SetWireGuardPeerSpec(context.Background(), "", "wireguardpeer-abc", spec)
	if err != nil {
		t.Fatalf("SetWireGuardPeerSpec error: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotPath != "/wireguardpeers/wireguardpeer-abc" {
		t.Errorf("path = %q, want /wireguardpeers/wireguardpeer-abc", gotPath)
	}
}
