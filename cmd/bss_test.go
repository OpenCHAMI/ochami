// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// bss_test.go exercises the commands directly under "bss" (dumpstate,
// history, service status, and the deprecated status) end-to-end against an
// httptest.Server, asserting outbound request method/path/query and the exit
// codes that command errors resolve to. The "bss boot" and "bss hosts"
// commands are covered in their own bss_<command>_test.go files, and
// HTTP-failure cases in bss_errors_test.go.

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestBSSDumpstate verifies "bss dumpstate" issues GET /dumpstate.
func TestBSSDumpstate(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Write([]byte(`{"state":"ok"}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "dumpstate", "--ignore-config", "--uri", srv.URL)

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/dumpstate" {
		t.Errorf("path = %q, want /dumpstate", gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
}

// TestBSSServiceStatus_Success verifies "bss service status" issues GET /service/status.
func TestBSSServiceStatus_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "service", "status", "--ignore-config", "--uri", srv.URL)

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/service/status" {
		t.Errorf("path = %q, want /service/status", gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
}

// TestBSSHistoryGet_Success verifies "bss history" issues GET /endpoint-history.
func TestBSSHistoryGet_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "history", "--ignore-config", "--uri", srv.URL)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/endpoint-history" {
		t.Errorf("path = %q, want /endpoint-history", gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
}

// TestBSSStatus_Deprecated verifies the deprecated top-level "bss status" command
// still issues a GET under /service.
func TestBSSStatus_Deprecated(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "status", "--ignore-config", "--uri", srv.URL)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.HasPrefix(gotPath, "/service") {
		t.Errorf("path = %q, want a /service path", gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
}

// TestBSSHistoryGet_QueryAndFormats verifies the history query builder and
// output-format variants.
func TestBSSHistoryGet_QueryAndFormats(t *testing.T) {
	t.Parallel()

	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Write([]byte(`[{"ID":"x0c0s0b0n0"}]`))
	}))
	defer srv.Close()

	for _, f := range []string{"json", "yaml"} {
		res := runOchamiWithRuntime(t, "bss", "history", "--ignore-config", "--uri", srv.URL,
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
