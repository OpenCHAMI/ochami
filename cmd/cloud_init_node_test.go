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
)

// TestCloudInitNodeGetData_Success verifies the per-datatype node get subcommands issue
// GET requests under /admin/impersonation/<id>/<datatype>.
func TestCloudInitNodeGetData_Success(t *testing.T) {
	t.Parallel()

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
			var gotMethod, gotPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotMethod = r.Method
				w.Write([]byte("#cloud-config\n"))
			}))
			defer srv.Close()

			t.Parallel()
			res := runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "node", "get", tt.sub, "x3000c0s0b0n0",
				"--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if !strings.HasPrefix(gotPath, "/admin/impersonation/x3000c0s0b0n0") {
				t.Errorf("path = %q, want it under /admin/impersonation/x3000c0s0b0n0", gotPath)
			}
			if gotMethod != http.MethodGet {
				t.Errorf("method = %q, want GET", gotMethod)
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
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Write([]byte("#cloud-config\n"))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "node", "get", "group", "x3000c0s0b0n0", "compute",
		"--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.HasPrefix(gotPath, "/admin/impersonation/x3000c0s0b0n0") {
		t.Errorf("path = %q, want it under the impersonation path", gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if !strings.Contains(gotPath, "compute") {
		t.Errorf("path = %q, want it to reference the compute group", gotPath)
	}
}

// TestCloudInitNodeGet_MetadataFormats verifies the output-format variants of
// "node get meta-data".
func TestCloudInitNodeGet_MetadataFormats(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hostname: node01\n"))
	}))
	defer srv.Close()

	for _, f := range []string{"json", "json-pretty", "yaml"} {
		res := runOchamiWithRuntime(t, "cloud-init", "node", "get", "meta-data", "--ignore-config",
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
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("#cloud-config\nfoo: bar\n"))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "node", "get", "user-data", "--ignore-config",
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
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("#cloud-config\nvendor: acme\n"))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "node", "get", "vendor-data", "--ignore-config",
		"--uri", srv.URL, "--token", "t", "x0c0s0b0n0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "vendor: acme") {
		t.Errorf("stdout = %q, want the vendor-data content", res.stdout)
	}
}

// TestCloudInitNodeSet_Stdin verifies "node set" reads payload from stdin when -d
// is not supplied.
func TestCloudInitNodeSet_Stdin(t *testing.T) {
	t.Parallel()

	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInputAndRuntime(t, `[{"id":"x0c0s0b0n0"}]`,
		"cloud-init", "node", "set", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
}

// TestCloudInitNodeGet_DataHeaderModes verifies that, for two nodes,
// "cloud-init node get user-data" and "vendor-data" print a header above each
// node's data with --headers always or multiple, and none with --headers never.
func TestCloudInitNodeGet_DataHeaderModes(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("#cloud-config\nfoo: bar\n"))
	}))
	defer srv.Close()

	for _, sub := range []string{"user-data", "vendor-data"} {
		for _, mode := range []string{"always", "never", "multiple"} {
			res := runOchamiWithRuntime(t, "cloud-init", "node", "get", sub, "--ignore-config",
				"--uri", srv.URL, "--token", "t", "--headers", mode, "x0c0s0b0n0", "x0c0s0b0n1")
			if res.err != nil {
				t.Fatalf("%s headers=%s: unexpected error: %v (exit %d)", sub, mode, res.err, res.exitCode)
			}
			hasHeader := strings.Contains(res.stdout, "--- (1/2) node=x0c0s0b0n0")
			if wantHeader := mode != "never"; hasHeader != wantHeader {
				t.Errorf("%s headers=%s: output = %q, header shown = %v, want %v", sub, mode, res.stdout, hasHeader, wantHeader)
			}
		}
	}
}

// TestCloudInitNodeGet_GroupHeaderModes verifies that, for two groups,
// "cloud-init node get group" prints a header above each group's data with
// --headers always or multiple, and none with --headers never.
func TestCloudInitNodeGet_GroupHeaderModes(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("#cloud-config\nfoo: bar\n"))
	}))
	defer srv.Close()

	for _, mode := range []string{"always", "never", "multiple"} {
		res := runOchamiWithRuntime(t, "cloud-init", "node", "get", "group", "--ignore-config",
			"--uri", srv.URL, "--token", "t", "--headers", mode, "x0c0s0b0n0", "compute", "storage")
		if res.err != nil {
			t.Fatalf("headers=%s: unexpected error: %v (exit %d)", mode, res.err, res.exitCode)
		}
		if hasHeader, wantHeader := strings.Contains(res.stdout, "group=compute"), mode != "never"; hasHeader != wantHeader {
			t.Errorf("headers=%s: output = %q, header shown = %v, want %v", mode, res.stdout, hasHeader, wantHeader)
		}
	}
}

// TestCloudInitNodeGet_GroupSkipsEmptyGroup verifies that when one of several
// requested groups' cloud-config comes back empty, it is omitted from the
// rendered output (with a warning logged) rather than printed as a blank
// entry, while a populated group's data still appears.
func TestCloudInitNodeGet_GroupSkipsEmptyGroup(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "empty.yaml") {
			// No body written: this group's cloud-config is empty.
			return
		}
		w.Write([]byte("#cloud-config\nrole: worker\n"))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "node", "get", "group",
		"--uri", srv.URL, "--token", "t", "x0c0s0b0n0", "compute", "empty")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "role: worker") {
		t.Errorf("stdout = %q, want it to contain the populated group's data", res.stdout)
	}
	if strings.Contains(res.stdout, "group=empty") {
		t.Errorf("stdout = %q, want the empty group omitted from rendered output", res.stdout)
	}
	if !strings.Contains(res.stdout, "cloud-config for group empty was empty") {
		t.Errorf("output = %q, want a warning about the empty group", res.stdout)
	}
}
