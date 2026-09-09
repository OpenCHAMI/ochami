// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package boot_service

import (
	"context"
	"net/http"
	"testing"

	api "github.com/openchami/boot-service/apis/boot.openchami.io/v1"
	boot_service_client "github.com/openchami/boot-service/pkg/client"
	"github.com/openchami/fabrica/pkg/fabrica"
)

// TestAddBootConfigSpecs_SendsNameAndSpecWithoutEnvelopeExtras verifies the simple API omits envelope metadata.
func TestAddBootConfigSpecs_SendsNameAndSpecWithoutEnvelopeExtras(t *testing.T) {
	var gotBody map[string]interface{}
	var gotPath, gotMethod string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		decodeJSONBody(t, r, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		encodeJSONResponse(t, w, api.BootConfiguration{})
	})
	defer srv.Close()

	cfgs := []BootConfigSpec{
		{
			Name:                  "compute-boot",
			BootConfigurationSpec: api.BootConfigurationSpec{Hosts: []string{"host1"}},
		},
	}

	results := c.AddBootConfigSpecs(context.Background(), "", cfgs)
	for _, e := range results.Errors() {
		if e != nil {
			t.Fatalf("AddBootConfigSpecs per-request error: %v", e)
		}
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/bootconfigurations" {
		t.Errorf("path = %q, want /bootconfigurations", gotPath)
	}
	// Simple API still sends an envelope built from name + spec, but must NOT
	// carry labels/annotations supplied on the request.
	if _, ok := gotBody["labels"]; ok {
		t.Errorf("simple request unexpectedly included labels: %+v", gotBody["labels"])
	}
	meta, _ := gotBody["metadata"].(map[string]interface{})
	if meta == nil || meta["name"] != "compute-boot" {
		t.Errorf("metadata.name = %+v, want compute-boot", meta)
	}
}

// TestAddBootConfigs_EnvelopeIncludesLabels verifies the advanced API preserves resource labels.
func TestAddBootConfigs_EnvelopeIncludesLabels(t *testing.T) {
	var gotBody map[string]interface{}
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		decodeJSONBody(t, r, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		encodeJSONResponse(t, w, api.BootConfiguration{})
	})
	defer srv.Close()

	cfgs := []boot_service_client.CreateBootConfigurationRequest{
		{
			Metadata: fabrica.Metadata{Name: "compute-boot", Labels: map[string]string{"env": "prod"}},
			Spec:     api.BootConfigurationSpec{Hosts: []string{"host1"}},
			Labels:   map[string]string{"env": "prod"},
		},
	}

	if results := c.AddBootConfigs(context.Background(), "", cfgs); results.HasErrors() {
		t.Fatalf("AddBootConfigs returned errors: %v", results.Errors())
	}

	labels, ok := gotBody["labels"].(map[string]interface{})
	if !ok || labels["env"] != "prod" {
		t.Errorf("envelope request labels = %+v, want env=prod", gotBody["labels"])
	}
}

// TestAddBootConfigSpecs_ReturnsOnlyCreatedResources verifies Values excludes failed simple API requests.
func TestAddBootConfigSpecs_ReturnsOnlyCreatedResources(t *testing.T) {
	requests := 0
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			http.Error(w, "creation failed", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		encodeJSONResponse(t, w, api.BootConfiguration{Metadata: fabrica.Metadata{Name: "created config"}})
	})
	defer srv.Close()

	results := c.AddBootConfigSpecs(context.Background(), "", []BootConfigSpec{
		{Name: "failed config"},
		{Name: "created config"},
	})
	if len(results.Errors()) != 1 {
		t.Fatalf("got %d per-request errors, want 1", len(results.Errors()))
	}
	created := results.Values()
	if len(created) != 1 {
		t.Fatalf("got %d created boot configurations, want 1", len(created))
	}
	if created[0] == nil || created[0].Metadata.Name != "created config" {
		t.Errorf("created boot configurations = %+v, want only created config", created)
	}
}

// TestAddBootConfigs_ReturnsOnlyCreatedResources verifies Values excludes failed advanced API requests.
func TestAddBootConfigs_ReturnsOnlyCreatedResources(t *testing.T) {
	requests := 0
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			http.Error(w, "creation failed", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		encodeJSONResponse(t, w, api.BootConfiguration{Metadata: fabrica.Metadata{Name: "created config"}})
	})
	defer srv.Close()

	results := c.AddBootConfigs(context.Background(), "", []boot_service_client.CreateBootConfigurationRequest{
		{Metadata: fabrica.Metadata{Name: "failed config"}},
		{Metadata: fabrica.Metadata{Name: "created config"}},
	})
	if len(results.Errors()) != 1 {
		t.Fatalf("got %d per-request errors, want 1", len(results.Errors()))
	}
	created := results.Values()
	if len(created) != 1 {
		t.Fatalf("got %d created boot configurations, want 1", len(created))
	}
	if created[0] == nil || created[0].Metadata.Name != "created config" {
		t.Errorf("created boot configurations = %+v, want only created config", created)
	}
}

// TestSetBootConfigSpec_SendsSpecToUIDEndpoint verifies simple updates target the requested resource UID.
func TestSetBootConfigSpec_SendsSpecToUIDEndpoint(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]interface{}
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		decodeJSONBody(t, r, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		encodeJSONResponse(t, w, api.BootConfiguration{})
	})
	defer srv.Close()

	spec := api.BootConfigurationSpec{Hosts: []string{"host1"}}

	_, err := c.SetBootConfigSpec(context.Background(), "", "boo-abc123", spec)
	if err != nil {
		t.Fatalf("SetBootConfigSpec returned error: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotPath != "/bootconfigurations/boo-abc123" {
		t.Errorf("path = %q, want /bootconfigurations/boo-abc123", gotPath)
	}
	if _, ok := gotBody["labels"]; ok {
		t.Errorf("simple set unexpectedly included labels: %+v", gotBody["labels"])
	}
}
