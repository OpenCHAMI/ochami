// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// cloud_init_node_branches_test.go covers the branch families of the
// "cloud-init node" verbs (get meta-data/user-data/vendor-data/group, set) that
// the happy paths do not reach: output-format variants, per-item HTTP error
// aggregation, network error mapping, and the set stdin/HTTP-error paths.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestCloudInitNodeGetMetadataFormats verifies the output-format variants of
// "node get meta-data".
func TestCloudInitNodeGetMetadataFormats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hostname: node01\n"))
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

// TestCloudInitNodeGetUserdata verifies "node get user-data" prints the raw
// user-data for the node.
func TestCloudInitNodeGetUserdata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("#cloud-config\nfoo: bar\n"))
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

// TestCloudInitNodeGetVendordata verifies "node get vendor-data" prints the raw
// vendor-data for the node.
func TestCloudInitNodeGetVendordata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("#cloud-config\nvendor: acme\n"))
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

// TestCloudInitNodeGetMetadataHTTPError verifies a failing meta-data fetch
// resolves to CodeHTTP via the aggregate.
func TestCloudInitNodeGetMetadataHTTPError(t *testing.T) {
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

// TestCloudInitNodeGetUserdataHTTPError verifies a failing user-data fetch
// resolves to CodeHTTP.
func TestCloudInitNodeGetUserdataHTTPError(t *testing.T) {
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

// TestCloudInitNodeGetGroupHTTPError verifies a failing node-group fetch
// resolves to CodeHTTP via the aggregate.
func TestCloudInitNodeGetGroupHTTPError(t *testing.T) {
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

// TestCloudInitNodeSetStdin verifies "node set" reads payload from stdin when -d
// is not supplied.
func TestCloudInitNodeSetStdin(t *testing.T) {
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

// TestCloudInitNodeSetHTTPError verifies a failing "node set" resolves to
// CodeHTTP via the aggregate.
func TestCloudInitNodeSetHTTPError(t *testing.T) {
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

// TestCloudInitNodeSetMalformedPayload verifies malformed inline payload
// resolves to CodePayload.
func TestCloudInitNodeSetMalformedPayload(t *testing.T) {
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

// TestCloudInitNodeGetDataHeaderModes verifies the --headers always/never modes
// for user-data and vendor-data over multiple nodes (exercising the
// header-printing arms).
func TestCloudInitNodeGetDataHeaderModes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("#cloud-config\nfoo: bar\n"))
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

// TestCloudInitNodeGetGroupHeaderModes verifies the --headers modes for the node
// group data over multiple groups.
func TestCloudInitNodeGetGroupHeaderModes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("#cloud-config\nfoo: bar\n"))
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
