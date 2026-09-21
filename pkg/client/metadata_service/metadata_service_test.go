// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

// metadata_service_test.go covers the read-only MetadataServiceClient methods (GetHealth,
// and List/Get for groups, defaults, instance-infos, and WireGuard peers),
// asserting request routing against an httptest.Server. Write-path methods are
// covered in the resource-specific *_test.go files.

import (
	"context"
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

	if _, err := c.GetHealth(context.Background(), format.DataFormatJson); err != nil {
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
		{"groups", func(c *MetadataServiceClient) error {
			_, e := c.ListGroups(context.Background(), "", format.DataFormatJson)
			return e
		}, "/groups"},
		{"defaults", func(c *MetadataServiceClient) error {
			_, e := c.ListDefaults(context.Background(), "", format.DataFormatJson)
			return e
		}, "/clusterdefaultss"},
		{"instanceinfos", func(c *MetadataServiceClient) error {
			_, e := c.ListInstanceInfos(context.Background(), "", format.DataFormatJson)
			return e
		}, "/instanceinfos"},
		{"wireguardpeers", func(c *MetadataServiceClient) error {
			_, e := c.ListWireGuardPeers(context.Background(), "", format.DataFormatJson)
			return e
		}, "/wireguardpeers"},
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
		call     func(c *MetadataServiceClient) error
		wantPath string
	}{
		{"group", func(c *MetadataServiceClient) error {
			_, e := c.GetGroup(context.Background(), "", format.DataFormatJson, "uid1")
			return e
		}, "/groups/uid1"},
		{"defaults", func(c *MetadataServiceClient) error {
			_, e := c.GetDefaults(context.Background(), "", format.DataFormatJson, "uid1")
			return e
		}, "/clusterdefaultss/uid1"},
		{"instanceinfo", func(c *MetadataServiceClient) error {
			_, e := c.GetInstanceInfo(context.Background(), "", format.DataFormatJson, "uid1")
			return e
		}, "/instanceinfos/uid1"},
		{"wireguardpeer", func(c *MetadataServiceClient) error {
			_, e := c.GetWireGuardPeer(context.Background(), "", format.DataFormatJson, "uid1")
			return e
		}, "/wireguardpeers/uid1"},
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

func TestReadEndpointsRejectUnsupportedOutputFormat(t *testing.T) {
	tests := []struct {
		name string
		body string
		call func(*MetadataServiceClient) error
	}{
		{name: "get group", body: `{}`, call: func(c *MetadataServiceClient) error {
			_, err := c.GetGroup(context.Background(), "", format.DataFormat("toml"), "uid")
			return err
		}},
		{name: "list groups", body: `[]`, call: func(c *MetadataServiceClient) error {
			_, err := c.ListGroups(context.Background(), "", format.DataFormat("toml"))
			return err
		}},
		{name: "get defaults", body: `{}`, call: func(c *MetadataServiceClient) error {
			_, err := c.GetDefaults(context.Background(), "", format.DataFormat("toml"), "uid")
			return err
		}},
		{name: "list defaults", body: `[]`, call: func(c *MetadataServiceClient) error {
			_, err := c.ListDefaults(context.Background(), "", format.DataFormat("toml"))
			return err
		}},
		{name: "get instance info", body: `{}`, call: func(c *MetadataServiceClient) error {
			_, err := c.GetInstanceInfo(context.Background(), "", format.DataFormat("toml"), "uid")
			return err
		}},
		{name: "list instance infos", body: `[]`, call: func(c *MetadataServiceClient) error {
			_, err := c.ListInstanceInfos(context.Background(), "", format.DataFormat("toml"))
			return err
		}},
		{name: "get wireguard peer", body: `{}`, call: func(c *MetadataServiceClient) error {
			_, err := c.GetWireGuardPeer(context.Background(), "", format.DataFormat("toml"), "uid")
			return err
		}},
		{name: "list wireguard peers", body: `[]`, call: func(c *MetadataServiceClient) error {
			_, err := c.ListWireGuardPeers(context.Background(), "", format.DataFormat("toml"))
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body)) //nolint:errcheck // client observes the response
			})
			defer srv.Close()
			if err := tt.call(c); err == nil {
				t.Fatal("call returned nil error for unsupported output format")
			}
		})
	}
}
