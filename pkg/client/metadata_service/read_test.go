// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

// read_test.go covers the read-only MetadataServiceClient methods (GetHealth,
// and List/Get for groups, defaults, instance-infos, and WireGuard peers),
// asserting request routing against an httptest.Server. Write-path methods are
// covered in the resource-specific *_test.go files.

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
		_, _ = w.Write([]byte(`{}`))
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
		call     func(c *MetadataServiceClient) error
		wantPath string
	}{
		{"groups", func(c *MetadataServiceClient) error { _, e := c.ListGroups("", format.DataFormatJson); return e }, "/groups"},
		{"defaults", func(c *MetadataServiceClient) error { _, e := c.ListDefaults("", format.DataFormatJson); return e }, "/clusterdefaultss"},
		{"instanceinfos", func(c *MetadataServiceClient) error { _, e := c.ListInstanceInfos("", format.DataFormatJson); return e }, "/instanceinfos"},
		{"wireguardpeers", func(c *MetadataServiceClient) error {
			_, e := c.ListWireGuardPeers("", format.DataFormatJson)
			return e
		}, "/wireguardpeers"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotPath string
			c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[]`))
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
		call     func(c *MetadataServiceClient) error
		wantPath string
	}{
		{"group", func(c *MetadataServiceClient) error { _, e := c.GetGroup("", format.DataFormatJson, "uid1"); return e }, "/groups/uid1"},
		{"defaults", func(c *MetadataServiceClient) error {
			_, e := c.GetDefaults("", format.DataFormatJson, "uid1")
			return e
		}, "/clusterdefaultss/uid1"},
		{"instanceinfo", func(c *MetadataServiceClient) error {
			_, e := c.GetInstanceInfo("", format.DataFormatJson, "uid1")
			return e
		}, "/instanceinfos/uid1"},
		{"wireguardpeer", func(c *MetadataServiceClient) error {
			_, e := c.GetWireGuardPeer("", format.DataFormatJson, "uid1")
			return e
		}, "/wireguardpeers/uid1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotMethod, gotPath string
			c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{}`))
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
