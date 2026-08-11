// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/openchami/fabrica/pkg/fabrica"
	api "github.com/openchami/metadata-service/apis/cloud-init.openchami.io/v1"
	metadata_service_client "github.com/openchami/metadata-service/pkg/client"
)

func TestAddWireGuardPeerSpecs_OmitsLabels(t *testing.T) {
	var gotBody map[string]interface{}
	var gotPath, gotMethod string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.WireGuardPeer{})
	})
	defer srv.Close()

	peers := []WireGuardPeerSpec{
		{
			Name:              "peer-nid001000",
			WireGuardPeerSpec: api.WireGuardPeerSpec{PublicKey: "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=", AllowedIP: "10.42.1.1/32"},
		},
	}

	_, errs, err := c.AddWireGuardPeerSpecs("", peers)
	if err != nil {
		t.Fatalf("AddWireGuardPeerSpecs func error: %v", err)
	}
	for _, e := range errs {
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

func TestAddWireGuardPeers_EnvelopeIncludesLabels(t *testing.T) {
	var gotBody map[string]interface{}
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.WireGuardPeer{})
	})
	defer srv.Close()

	peers := []metadata_service_client.CreateWireGuardPeerRequest{
		{
			Metadata: fabrica.Metadata{Name: "peer-nid001000", Labels: map[string]string{"env": "prod"}},
			Spec:     api.WireGuardPeerSpec{PublicKey: "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=", AllowedIP: "10.42.1.1/32"},
			Labels:   map[string]string{"env": "prod"},
		},
	}

	_, _, err := c.AddWireGuardPeers("", peers)
	if err != nil {
		t.Fatalf("AddWireGuardPeers func error: %v", err)
	}

	labels, ok := gotBody["labels"].(map[string]interface{})
	if !ok || labels["env"] != "prod" {
		t.Errorf("envelope request labels = %+v, want env=prod", gotBody["labels"])
	}
}

func TestSetWireGuardPeerSpec_UsesUIDEndpoint(t *testing.T) {
	var gotPath, gotMethod string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.WireGuardPeer{})
	})
	defer srv.Close()

	spec := api.WireGuardPeerSpec{PublicKey: "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=", AllowedIP: "10.42.1.1/32"}

	_, err := c.SetWireGuardPeerSpec("", "wireguardpeer-abc", spec)
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
