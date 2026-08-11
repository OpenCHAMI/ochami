// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package boot_service

import (
	"encoding/json"
	"net/http"
	"testing"

	api "github.com/openchami/boot-service/apis/boot.openchami.io/v1"
	boot_service_client "github.com/openchami/boot-service/pkg/client"
	"github.com/openchami/fabrica/pkg/fabrica"
)

func TestAddBMCSpecs_SendsNameAndSpecWithoutEnvelopeExtras(t *testing.T) {
	var gotBody map[string]interface{}
	var gotPath, gotMethod string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.BMC{})
	})
	defer srv.Close()

	bmcs := []BMCSpec{
		{
			Name:    "bmc01",
			BMCSpec: api.BMCSpec{XName: "x1000c0s0b0"},
		},
	}

	_, errs, err := c.AddBMCSpecs("", bmcs)
	if err != nil {
		t.Fatalf("AddBMCSpecs returned func error: %v", err)
	}
	for _, e := range errs {
		if e != nil {
			t.Fatalf("AddBMCSpecs per-request error: %v", e)
		}
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/bmcs" {
		t.Errorf("path = %q, want /bmcs", gotPath)
	}
	// Simple API still sends an envelope built from name + spec, but must NOT
	// carry labels/annotations supplied on the request.
	if _, ok := gotBody["labels"]; ok {
		t.Errorf("simple request unexpectedly included labels: %+v", gotBody["labels"])
	}
	meta, _ := gotBody["metadata"].(map[string]interface{})
	if meta == nil || meta["name"] != "bmc01" {
		t.Errorf("metadata.name = %+v, want bmc01", meta)
	}
}

func TestAddBMCs_EnvelopeIncludesLabels(t *testing.T) {
	var gotBody map[string]interface{}
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.BMC{})
	})
	defer srv.Close()

	bmcs := []boot_service_client.CreateBMCRequest{
		{
			Metadata: fabrica.Metadata{Name: "bmc01", Labels: map[string]string{"env": "prod"}},
			Spec:     api.BMCSpec{XName: "x1000c0s0b0"},
			Labels:   map[string]string{"env": "prod"},
		},
	}

	_, _, err := c.AddBMCs("", bmcs)
	if err != nil {
		t.Fatalf("AddBMCs returned func error: %v", err)
	}

	labels, ok := gotBody["labels"].(map[string]interface{})
	if !ok || labels["env"] != "prod" {
		t.Errorf("envelope request labels = %+v, want env=prod", gotBody["labels"])
	}
}

func TestAddBMCSpecs_ReturnsOnlyCreatedResources(t *testing.T) {
	requests := 0
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			http.Error(w, "creation failed", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.BMC{Metadata: fabrica.Metadata{Name: "created BMC"}})
	})
	defer srv.Close()

	created, errs, err := c.AddBMCSpecs("", []BMCSpec{
		{Name: "failed BMC"},
		{Name: "created BMC"},
	})
	if err != nil {
		t.Fatalf("AddBMCSpecs returned func error: %v", err)
	}
	if len(errs) != 1 {
		t.Fatalf("got %d per-request errors, want 1", len(errs))
	}
	if len(created) != 1 {
		t.Fatalf("got %d created BMCs, want 1", len(created))
	}
	if created[0] == nil || created[0].Metadata.Name != "created BMC" {
		t.Errorf("created BMCs = %+v, want only created BMC", created)
	}
}

func TestAddBMCs_ReturnsOnlyCreatedResources(t *testing.T) {
	requests := 0
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			http.Error(w, "creation failed", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.BMC{Metadata: fabrica.Metadata{Name: "created BMC"}})
	})
	defer srv.Close()

	created, errs, err := c.AddBMCs("", []boot_service_client.CreateBMCRequest{
		{Metadata: fabrica.Metadata{Name: "failed BMC"}},
		{Metadata: fabrica.Metadata{Name: "created BMC"}},
	})
	if err != nil {
		t.Fatalf("AddBMCs returned func error: %v", err)
	}
	if len(errs) != 1 {
		t.Fatalf("got %d per-request errors, want 1", len(errs))
	}
	if len(created) != 1 {
		t.Fatalf("got %d created BMCs, want 1", len(created))
	}
	if created[0] == nil || created[0].Metadata.Name != "created BMC" {
		t.Errorf("created BMCs = %+v, want only created BMC", created)
	}
}

func TestSetBMCSpec_SendsSpecToUIDEndpoint(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]interface{}
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.BMC{})
	})
	defer srv.Close()

	spec := api.BMCSpec{XName: "x1000c0s0b0"}

	_, err := c.SetBMCSpec("", "bmc-abc123", spec)
	if err != nil {
		t.Fatalf("SetBMCSpec returned error: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotPath != "/bmcs/bmc-abc123" {
		t.Errorf("path = %q, want /bmcs/bmc-abc123", gotPath)
	}
	if _, ok := gotBody["labels"]; ok {
		t.Errorf("simple set unexpectedly included labels: %+v", gotBody["labels"])
	}
}
