// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// bss_boot_script_test.go exercises "bss boot script get" end-to-end against
// an httptest.Server, asserting the outbound request and its query string.
// HTTP-failure cases are covered in bss_boot_script_errors_test.go.

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestBSSBootScriptGet_Success verifies "bss boot script get" issues GET /bootscript
// with the selector encoded in the query string.
func TestBSSBootScriptGet_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		gotMethod = r.Method
		w.Write([]byte(`#!ipxe`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "boot", "script", "get",
		"--ignore-config", "--uri", srv.URL, "--mac", "de:ad:be:ef:00:00")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/bootscript" {
		t.Errorf("path = %q, want /bootscript", gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if !strings.Contains(gotQuery, "mac=de%3Aad%3Abe%3Aef%3A00%3A00") {
		t.Errorf("query = %q, want it to contain the URL-encoded mac", gotQuery)
	}
}

// TestBSSBootScriptGet_Query verifies the boot-script query builder emits the
// xname (as name) and the retry, arch, and timestamp parameters.
func TestBSSBootScriptGet_Query(t *testing.T) {
	t.Parallel()

	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Write([]byte(`#!ipxe`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "boot", "script", "get", "--ignore-config", "--uri", srv.URL,
		"--xname", "x0c0s0b0n0", "--retry", "3", "--arch", "x86_64", "--timestamp", "12345")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	for key, want := range map[string]string{"name": "x0c0s0b0n0", "retry": "3", "arch": "x86_64", "timestamp": "12345"} {
		if got := gotQuery.Get(key); got != want {
			t.Errorf("query %s = %q, want %q", key, got, want)
		}
	}
}
