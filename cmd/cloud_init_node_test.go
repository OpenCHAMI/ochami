// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// cloud_init_node_test.go exercises the "cloud-init node get" subcommands
// (meta-data, user-data, vendor-data, group) end-to-end against an
// httptest.Server, asserting the impersonation path and exit codes.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestCloudInitNodeGetData verifies the per-datatype node get subcommands issue
// GET requests under /admin/impersonation/<id>/<datatype>.
func TestCloudInitNodeGetData(t *testing.T) {
	tests := []struct {
		sub      string
		wantLeaf string
	}{
		{"meta-data", "meta-data"},
		{"user-data", "user-data"},
		{"vendor-data", "vendor-data"},
	}
	for _, tt := range tests {
		t.Run(tt.sub, func(t *testing.T) {
			var gotPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				_, _ = w.Write([]byte("#cloud-config\n")) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchami(t, "cloud-init", "node", "get", tt.sub, "x3000c0s0b0n0",
				"--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if !strings.HasPrefix(gotPath, "/admin/impersonation/x3000c0s0b0n0") {
				t.Errorf("path = %q, want it under /admin/impersonation/x3000c0s0b0n0", gotPath)
			}
			if !strings.HasSuffix(gotPath, tt.wantLeaf) {
				t.Errorf("path = %q, want it to end with %q", gotPath, tt.wantLeaf)
			}
		})
	}
}

// TestCloudInitNodeGetGroup verifies "cloud-init node get group" issues a GET
// under the impersonation path for the given group.
func TestCloudInitNodeGetGroup(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte("#cloud-config\n")) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "node", "get", "group", "x3000c0s0b0n0", "compute",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.HasPrefix(gotPath, "/admin/impersonation/x3000c0s0b0n0") {
		t.Errorf("path = %q, want it under the impersonation path", gotPath)
	}
	if !strings.Contains(gotPath, "compute") {
		t.Errorf("path = %q, want it to reference the compute group", gotPath)
	}
}

// TestCloudInitNodeGetDataHTTPError verifies an unsuccessful HTTP response from
// a node get resolves to CodeHTTP.
func TestCloudInitNodeGetDataHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "node", "get", "meta-data", "x3000c0s0b0n0",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}
