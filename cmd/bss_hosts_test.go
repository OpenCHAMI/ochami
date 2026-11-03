// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// bss_hosts_test.go exercises "bss hosts get" end-to-end against an
// httptest.Server, asserting the outbound request, its query string, and the
// output formats. HTTP-failure cases are covered in bss_hosts_errors_test.go.

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// TestBSSHostsGet_Success verifies "bss hosts get" issues GET /hosts.
func TestBSSHostsGet_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "hosts", "get", "--ignore-config", "--uri", srv.URL)

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/hosts" {
		t.Errorf("path = %q, want /hosts", gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
}

// TestBSSHostsGet_QueryAndFormats verifies the hosts query builder and
// output-format variants.
func TestBSSHostsGet_QueryAndFormats(t *testing.T) {
	t.Parallel()

	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Write([]byte(`[{"ID":"x0c0s0b0n0"}]`))
	}))
	defer srv.Close()

	for _, f := range []string{"json", "yaml"} {
		res := runOchamiWithRuntime(t, "bss", "hosts", "get", "--ignore-config", "--uri", srv.URL,
			"--xname", "x0c0s0b0n0", "-F", f)
		if res.err != nil {
			t.Fatalf("format %s: unexpected error: %v (exit %d)", f, res.err, res.exitCode)
		}
		assertFormattedOutput(t, f, res.stdout, "ID", "x0c0s0b0n0")
	}
	if len(gotQuery) == 0 {
		t.Error("expected a non-empty query for --xname")
	}
}
