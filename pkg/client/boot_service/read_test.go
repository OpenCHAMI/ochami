// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package boot_service

// read_test.go covers the read-only BootServiceClient methods (GetHealth, and
// List/Get for boot configurations, BMCs, and nodes), asserting request routing
// against an httptest.Server. Write-path methods are covered in the
// resource-specific *_test.go files.

import (
	"net/http"
	"testing"

	"github.com/openchami/ochami/pkg/format"
)

// TestGetHealth verifies GetHealth issues GET /health.
func TestGetHealth(t *testing.T) {
	var gotMethod, gotPath string
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	})
	defer srv.Close()

	if _, err := c.GetHealth(format.DataFormatJson); err != nil {
		t.Fatalf("GetHealth: %v", err)
	}
	if gotMethod != http.MethodGet || gotPath != "/health" {
		t.Errorf("request = %s %s, want GET /health", gotMethod, gotPath)
	}
}

// TestListEndpoints verifies the List methods issue GET to their collection
// endpoints.
func TestListEndpoints(t *testing.T) {
	cases := []struct {
		name     string
		call     func(c *BootServiceClient) error
		wantPath string
	}{
		{"bootconfigs", func(c *BootServiceClient) error { _, e := c.ListBootConfigs("", format.DataFormatJson); return e }, "/bootconfigurations"},
		{"bmcs", func(c *BootServiceClient) error { _, e := c.ListBMCs("", format.DataFormatJson); return e }, "/bmcs"},
		{"nodes", func(c *BootServiceClient) error { _, e := c.ListNodes("", format.DataFormatJson); return e }, "/nodes"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotPath string
			c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
			})
			defer srv.Close()

			if err := tc.call(c); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if gotMethod != http.MethodGet || gotPath != tc.wantPath {
				t.Errorf("request = %s %s, want GET %s", gotMethod, gotPath, tc.wantPath)
			}
		})
	}
}

// TestGetEndpoints verifies the Get-by-uid methods issue GET to
// <collection>/<uid>.
func TestGetEndpoints(t *testing.T) {
	cases := []struct {
		name     string
		call     func(c *BootServiceClient) error
		wantPath string
	}{
		{"bootconfig", func(c *BootServiceClient) error { _, e := c.GetBootConfig("", format.DataFormatJson, "uid1"); return e }, "/bootconfigurations/uid1"},
		{"bmc", func(c *BootServiceClient) error { _, e := c.GetBMC("", format.DataFormatJson, "uid1"); return e }, "/bmcs/uid1"},
		{"node", func(c *BootServiceClient) error { _, e := c.GetNode("", format.DataFormatJson, "uid1"); return e }, "/nodes/uid1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotPath string
			c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
			})
			defer srv.Close()

			if err := tc.call(c); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if gotMethod != http.MethodGet || gotPath != tc.wantPath {
				t.Errorf("request = %s %s, want GET %s", gotMethod, gotPath, tc.wantPath)
			}
		})
	}
}
