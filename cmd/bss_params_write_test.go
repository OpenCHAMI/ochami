// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// bss_params_write_test.go covers the "bss boot params set" (PUT) and
// "bss boot params update" (PATCH) verbs end-to-end against an httptest.Server.

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestBSSBootParamsSet_Success verifies "bss boot params set" issues PUT /bootparameters.
func TestBSSBootParamsSet_Success(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "set",
		"--ignore-config", "--uri", srv.URL, "--token", "faketoken",
		"--mac", "de:ad:be:ef:00:00", "--kernel", "https://example.com/vmlinuz")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotPath != "/bootparameters" {
		t.Errorf("path = %q, want /bootparameters", gotPath)
	}
}

// TestBSSBootParamsUpdate_Success verifies "bss boot params update" issues PATCH
// /bootparameters.
func TestBSSBootParamsUpdate_Success(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "update",
		"--ignore-config", "--uri", srv.URL, "--token", "faketoken",
		"--mac", "de:ad:be:ef:00:00", "--kernel", "https://example.com/vmlinuz")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("method = %q, want PATCH", gotMethod)
	}
	if gotPath != "/bootparameters" {
		t.Errorf("path = %q, want /bootparameters", gotPath)
	}
}
