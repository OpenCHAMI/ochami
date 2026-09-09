// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package boot_service

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	api "github.com/openchami/boot-service/apis/boot.openchami.io/v1"
	boot_service_client "github.com/openchami/boot-service/pkg/client"
	"github.com/openchami/fabrica/pkg/fabrica"
)

func decodeJSONBody(t *testing.T, r *http.Request, dst any) {
	t.Helper()
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		t.Errorf("decode request body: %v", err)
	}
}

func encodeJSONResponse(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Errorf("encode response body: %v", err)
	}
}

// TestAddBMCSpecs_SendsNameAndSpecWithoutEnvelopeExtras verifies the simple API omits envelope metadata.
func TestAddBMCSpecs_SendsNameAndSpecWithoutEnvelopeExtras(t *testing.T) {
	var gotBody map[string]interface{}
	var gotPath, gotMethod string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		decodeJSONBody(t, r, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		encodeJSONResponse(t, w, api.BMC{})
	})
	defer srv.Close()

	bmcs := []BMCSpec{
		{
			Name:    "bmc01",
			BMCSpec: api.BMCSpec{XName: "x1000c0s0b0"},
		},
	}

	results := c.AddBMCSpecs(context.Background(), "", bmcs)
	for _, e := range results.Errors() {
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

// TestAddBMCs_EnvelopeIncludesLabels verifies the advanced API preserves resource labels.
func TestAddBMCs_EnvelopeIncludesLabels(t *testing.T) {
	var gotBody map[string]interface{}
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		decodeJSONBody(t, r, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		encodeJSONResponse(t, w, api.BMC{})
	})
	defer srv.Close()

	bmcs := []boot_service_client.CreateBMCRequest{
		{
			Metadata: fabrica.Metadata{Name: "bmc01", Labels: map[string]string{"env": "prod"}},
			Spec:     api.BMCSpec{XName: "x1000c0s0b0"},
			Labels:   map[string]string{"env": "prod"},
		},
	}

	if results := c.AddBMCs(context.Background(), "", bmcs); results.HasErrors() {
		t.Fatalf("AddBMCs results = %v, want success", results)
	}

	labels, ok := gotBody["labels"].(map[string]interface{})
	if !ok || labels["env"] != "prod" {
		t.Errorf("envelope request labels = %+v, want env=prod", gotBody["labels"])
	}
}

// TestAddBMCSpecs_ReturnsOnlyCreatedResources verifies Values excludes failed simple API requests.
func TestAddBMCSpecs_ReturnsOnlyCreatedResources(t *testing.T) {
	requests := 0
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			http.Error(w, "creation failed", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		encodeJSONResponse(t, w, api.BMC{Metadata: fabrica.Metadata{Name: "created BMC"}})
	})
	defer srv.Close()

	results := c.AddBMCSpecs(context.Background(), "", []BMCSpec{
		{Name: "failed BMC"},
		{Name: "created BMC"},
	})
	if len(results.Errors()) != 1 {
		t.Fatalf("got %d per-request errors, want 1", len(results.Errors()))
	}
	created := results.Values()
	if len(created) != 1 {
		t.Fatalf("got %d created BMCs, want 1", len(created))
	}
	if created[0] == nil || created[0].Metadata.Name != "created BMC" {
		t.Errorf("created BMCs = %+v, want only created BMC", created)
	}
}

// TestAddBMCs_ReturnsOnlyCreatedResources verifies Values excludes failed advanced API requests.
func TestAddBMCs_ReturnsOnlyCreatedResources(t *testing.T) {
	requests := 0
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			http.Error(w, "creation failed", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		encodeJSONResponse(t, w, api.BMC{Metadata: fabrica.Metadata{Name: "created BMC"}})
	})
	defer srv.Close()

	results := c.AddBMCs(context.Background(), "", []boot_service_client.CreateBMCRequest{
		{Metadata: fabrica.Metadata{Name: "failed BMC"}},
		{Metadata: fabrica.Metadata{Name: "created BMC"}},
	})
	if len(results.Errors()) != 1 {
		t.Fatalf("got %d per-request errors, want 1", len(results.Errors()))
	}
	created := results.Values()
	if len(created) != 1 {
		t.Fatalf("got %d created BMCs, want 1", len(created))
	}
	if created[0] == nil || created[0].Metadata.Name != "created BMC" {
		t.Errorf("created BMCs = %+v, want only created BMC", created)
	}
}

// TestSetBMCSpec_SendsSpecToUIDEndpoint verifies simple updates target the requested resource UID.
func TestSetBMCSpec_SendsSpecToUIDEndpoint(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]interface{}
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		decodeJSONBody(t, r, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		encodeJSONResponse(t, w, api.BMC{})
	})
	defer srv.Close()

	spec := api.BMCSpec{XName: "x1000c0s0b0"}

	_, err := c.SetBMCSpec(context.Background(), "", "bmc-abc123", spec)
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

// TestEnvelopeSetMethods verifies advanced updates use the correct endpoint and bearer token.
func TestEnvelopeSetMethods(t *testing.T) {
	tests := []struct {
		name     string
		wantPath string
		call     func(*BootServiceClient) error
	}{
		{"bmc", "/bmcs/uid", func(c *BootServiceClient) error {
			_, err := c.SetBMC(context.Background(), "tok", "uid", boot_service_client.UpdateBMCRequest{})
			return err
		}},
		{"config", "/bootconfigurations/uid", func(c *BootServiceClient) error {
			_, err := c.SetBootConfig(context.Background(), "tok", "uid", boot_service_client.UpdateBootConfigurationRequest{})
			return err
		}},
		{"node", "/nodes/uid", func(c *BootServiceClient) error {
			_, err := c.SetNode(context.Background(), "tok", "uid", boot_service_client.UpdateNodeRequest{})
			return err
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotPath, gotAuth string
			c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath, gotAuth = r.Method, r.URL.Path, r.Header.Get("Authorization")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
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
