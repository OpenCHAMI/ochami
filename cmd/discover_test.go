// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// discover_test.go exercises "ochami discover static", which reads a discovery
// payload and populates SMD by POSTing components, redfish endpoints, and
// groups. Tests run against an httptest.Server standing in for SMD.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// discoveryPayload is a minimal valid static-discovery payload with one BMC and
// one node (member of one group and with one ethernet interface).
const discoveryPayload = `{
  "bmcs": [
    {"xname": "x1000c1s7b0", "mac": "de:ca:fc:0f:ee:ee", "ip": "172.16.0.101"}
  ],
  "nodes": [
    {
      "name": "node01",
      "nid": 1,
      "xname": "x1000c1s7b0n0",
      "bmc": "x1000c1s7b0",
      "groups": ["compute"],
      "interfaces": [
        {"mac_addr": "de:ad:be:ee:ee:f1", "ip_addrs": [{"name": "internal", "ip_addr": "172.16.0.1"}]}
      ]
    }
  ]
}`

// TestDiscoverStatic verifies "discover static -d <payload>" populates SMD,
// issuing POST requests for the discovered structures, and exits successfully
// when the server accepts them.
func TestDiscoverStatic(t *testing.T) {
	sawPost := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			sawPost = true
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "discover", "static", "-d", discoveryPayload,
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !sawPost {
		t.Error("expected at least one POST to SMD, got none")
	}
}

// TestDiscoverStaticOverwrite verifies the --overwrite path, which PUTs/PATCHes
// existing structures instead of only POSTing.
func TestDiscoverStaticOverwrite(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "discover", "static", "-d", discoveryPayload, "--overwrite",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if res.exitCode != cli.CodeSuccess {
		t.Errorf("exit code = %d, want %d (CodeSuccess)", res.exitCode, cli.CodeSuccess)
	}
}

// TestDiscoverStaticHTTPError verifies that when SMD rejects the discovery
// writes, "discover static" resolves to the CodeHTTP exit code (the "completed
// with errors" aggregate).
func TestDiscoverStaticHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "discover", "static", "-d", discoveryPayload,
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}
