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

// TestCloudInitNodeGetData_Success verifies the per-datatype node get subcommands issue
// GET requests under /admin/impersonation/<id>/<datatype>.
func TestCloudInitNodeGetData_Success(t *testing.T) {
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

// TestCloudInitNodeGet_Group verifies "cloud-init node get group" issues a GET
// under the impersonation path for the given group.
func TestCloudInitNodeGet_Group(t *testing.T) {
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

// TestCloudInitNodeGetData_HTTPError verifies an unsuccessful HTTP response from
// a node get resolves to CodeHTTP.
func TestCloudInitNodeGetData_HTTPError(t *testing.T) {
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

// TestCloudInitNodeGet_MetadataFormats verifies the output-format variants of
// "node get meta-data".
func TestCloudInitNodeGet_MetadataFormats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hostname: node01\n")) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	for _, f := range []string{"json", "json-pretty", "yaml"} {
		res := runOchami(t, "cloud-init", "node", "get", "meta-data", "--ignore-config",
			"--uri", srv.URL, "--token", "t", "-F", f, "x0c0s0b0n0")
		if res.err != nil {
			t.Fatalf("format %s: unexpected error: %v (exit %d)", f, res.err, res.exitCode)
		}
		if !strings.Contains(res.stdout, "node01") {
			t.Errorf("format %s: stdout = %q, want it to contain the hostname", f, res.stdout)
		}
	}
}

// TestCloudInitNodeGet_Userdata verifies "node get user-data" prints the raw
// user-data for the node.
func TestCloudInitNodeGet_Userdata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("#cloud-config\nfoo: bar\n")) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "node", "get", "user-data", "--ignore-config",
		"--uri", srv.URL, "--token", "t", "x0c0s0b0n0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "foo: bar") {
		t.Errorf("stdout = %q, want the user-data content", res.stdout)
	}
}

// TestCloudInitNodeGet_Vendordata verifies "node get vendor-data" prints the raw
// vendor-data for the node.
func TestCloudInitNodeGet_Vendordata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("#cloud-config\nvendor: acme\n")) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "node", "get", "vendor-data", "--ignore-config",
		"--uri", srv.URL, "--token", "t", "x0c0s0b0n0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "vendor: acme") {
		t.Errorf("stdout = %q, want the vendor-data content", res.stdout)
	}
}

// TestCloudInitNodeGet_MetadataHTTPError verifies a failing meta-data fetch
// resolves to CodeHTTP via the aggregate.
func TestCloudInitNodeGet_MetadataHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "node", "get", "meta-data", "--ignore-config",
		"--uri", srv.URL, "--token", "t", "x0c0s0b0n0")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestCloudInitNodeGet_UserdataHTTPError verifies a failing user-data fetch
// resolves to CodeHTTP.
func TestCloudInitNodeGet_UserdataHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "node", "get", "user-data", "--ignore-config",
		"--uri", srv.URL, "--token", "t", "x0c0s0b0n0")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestCloudInitNodeGet_GroupHTTPError verifies a failing node-group fetch
// resolves to CodeHTTP via the aggregate.
func TestCloudInitNodeGet_GroupHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "node", "get", "group", "--ignore-config",
		"--uri", srv.URL, "--token", "t", "x0c0s0b0n0", "compute")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestCloudInitNodeSet_Stdin verifies "node set" reads payload from stdin when -d
// is not supplied.
func TestCloudInitNodeSet_Stdin(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInput(t, `[{"id":"x0c0s0b0n0"}]`,
		"cloud-init", "node", "set", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
}

// TestCloudInitNodeSet_HTTPError verifies a failing "node set" resolves to
// CodeHTTP via the aggregate.
func TestCloudInitNodeSet_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "node", "set", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"-d", `[{"id":"x0c0s0b0n0"}]`)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestCloudInitNodeSet_MalformedPayload verifies malformed inline payload
// resolves to CodePayload.
func TestCloudInitNodeSet_MalformedPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "node", "set", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"-d", `not json`)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (CodePayload)", res.exitCode, cli.CodePayload)
	}
}

// TestCloudInitNodeGet_DataHeaderModes verifies the --headers always/never modes
// for user-data and vendor-data over multiple nodes (exercising the
// header-printing arms).
func TestCloudInitNodeGet_DataHeaderModes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("#cloud-config\nfoo: bar\n")) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	for _, sub := range []string{"user-data", "vendor-data"} {
		for _, mode := range []string{"always", "never", "multiple"} {
			res := runOchami(t, "cloud-init", "node", "get", sub, "--ignore-config",
				"--uri", srv.URL, "--token", "t", "--headers", mode, "x0c0s0b0n0", "x0c0s0b0n1")
			if res.err != nil {
				t.Fatalf("%s headers=%s: unexpected error: %v (exit %d)", sub, mode, res.err, res.exitCode)
			}
		}
	}
}

// TestCloudInitNodeGet_GroupHeaderModes verifies the --headers modes for the node
// group data over multiple groups.
func TestCloudInitNodeGet_GroupHeaderModes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("#cloud-config\nfoo: bar\n")) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	for _, mode := range []string{"always", "never", "multiple"} {
		res := runOchami(t, "cloud-init", "node", "get", "group", "--ignore-config",
			"--uri", srv.URL, "--token", "t", "--headers", mode, "x0c0s0b0n0", "compute", "storage")
		if res.err != nil {
			t.Fatalf("headers=%s: unexpected error: %v (exit %d)", mode, res.err, res.exitCode)
		}
	}
}
