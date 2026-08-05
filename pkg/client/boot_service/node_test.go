// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package boot_service

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	api "github.com/openchami/boot-service/apis/boot.openchami.io/v1"
	boot_service_client "github.com/openchami/boot-service/pkg/client"
	"github.com/openchami/fabrica/pkg/fabrica"
	"github.com/rs/zerolog"
)

// newTestClient spins up an httptest server with the given handler and returns
// a BootServiceClient pointed at it.
func newTestClient(t *testing.T, handler http.HandlerFunc) (*BootServiceClient, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	c, err := NewClient(srv.URL, false, 5*time.Second, "", zerolog.New(io.Discard))
	if err != nil {
		srv.Close()
		t.Fatalf("failed to create client: %v", err)
	}
	return c, srv
}

func TestAddNodeSpecs_SendsNameAndSpecWithoutEnvelopeExtras(t *testing.T) {
	var gotBody map[string]interface{}
	var gotPath, gotMethod string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.Node{})
	})
	defer srv.Close()

	nodes := []NodeSpec{
		{
			Name:     "x1000c0s0b0n0",
			NodeSpec: api.NodeSpec{XName: "x1000c0s0b0n0", NID: 42},
		},
	}

	_, errs, err := c.AddNodeSpecs("", nodes)
	if err != nil {
		t.Fatalf("AddNodeSpecs returned func error: %v", err)
	}
	for _, e := range errs {
		if e != nil {
			t.Fatalf("AddNodeSpecs per-request error: %v", e)
		}
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/nodes" {
		t.Errorf("path = %q, want /nodes", gotPath)
	}
	// Simple API still sends an envelope built from name + spec, but must NOT
	// carry labels/annotations supplied on the request.
	if _, ok := gotBody["labels"]; ok {
		t.Errorf("simple request unexpectedly included labels: %+v", gotBody["labels"])
	}
	meta, _ := gotBody["metadata"].(map[string]interface{})
	if meta == nil || meta["name"] != "x1000c0s0b0n0" {
		t.Errorf("metadata.name = %+v, want x1000c0s0b0n0", meta)
	}
}

func TestAddNodes_EnvelopeIncludesLabels(t *testing.T) {
	var gotBody map[string]interface{}
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.Node{})
	})
	defer srv.Close()

	nodes := []boot_service_client.CreateNodeRequest{
		{
			Metadata: fabrica.Metadata{Name: "x1000c0s0b0n0", Labels: map[string]string{"env": "prod"}},
			Spec:     api.NodeSpec{XName: "x1000c0s0b0n0"},
			Labels:   map[string]string{"env": "prod"},
		},
	}

	_, _, err := c.AddNodes("", nodes)
	if err != nil {
		t.Fatalf("AddNodes returned func error: %v", err)
	}

	labels, ok := gotBody["labels"].(map[string]interface{})
	if !ok || labels["env"] != "prod" {
		t.Errorf("envelope request labels = %+v, want env=prod", gotBody["labels"])
	}
}

func TestAddNodeSpecs_ReturnsOnlyCreatedResources(t *testing.T) {
	requests := 0
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			http.Error(w, "creation failed", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.Node{Metadata: fabrica.Metadata{Name: "created node"}})
	})
	defer srv.Close()

	created, errs, err := c.AddNodeSpecs("", []NodeSpec{
		{Name: "failed node"},
		{Name: "created node"},
	})
	if err != nil {
		t.Fatalf("AddNodeSpecs returned func error: %v", err)
	}
	if len(errs) != 1 {
		t.Fatalf("got %d per-request errors, want 1", len(errs))
	}
	if len(created) != 1 {
		t.Fatalf("got %d created nodes, want 1", len(created))
	}
	if created[0] == nil || created[0].Metadata.Name != "created node" {
		t.Errorf("created nodes = %+v, want only created node", created)
	}
}

func TestAddNodes_ReturnsOnlyCreatedResources(t *testing.T) {
	requests := 0
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			http.Error(w, "creation failed", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.Node{Metadata: fabrica.Metadata{Name: "created node"}})
	})
	defer srv.Close()

	created, errs, err := c.AddNodes("", []boot_service_client.CreateNodeRequest{
		{Metadata: fabrica.Metadata{Name: "failed node"}},
		{Metadata: fabrica.Metadata{Name: "created node"}},
	})
	if err != nil {
		t.Fatalf("AddNodes returned func error: %v", err)
	}
	if len(errs) != 1 {
		t.Fatalf("got %d per-request errors, want 1", len(errs))
	}
	if len(created) != 1 {
		t.Fatalf("got %d created nodes, want 1", len(created))
	}
	if created[0] == nil || created[0].Metadata.Name != "created node" {
		t.Errorf("created nodes = %+v, want only created node", created)
	}
}

func TestSetNodeSpec_SendsSpecToUIDEndpoint(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]interface{}
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.Node{})
	})
	defer srv.Close()

	spec := api.NodeSpec{XName: "x1000c0s0b0n0", NID: 7}

	_, err := c.SetNodeSpec("", "nod-abc123", spec)
	if err != nil {
		t.Fatalf("SetNodeSpec returned error: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotPath != "/nodes/nod-abc123" {
		t.Errorf("path = %q, want /nodes/nod-abc123", gotPath)
	}
	if _, ok := gotBody["labels"]; ok {
		t.Errorf("simple set unexpectedly included labels: %+v", gotBody["labels"])
	}
}
