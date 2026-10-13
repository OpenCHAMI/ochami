// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_compep_test.go covers the success paths of the "smd compep" verbs (get,
// delete): the get-all vs get-by-xname arms, output-format variants, and the
// delete --all/args/-d selection with confirmation. HTTP and network error
// mapping, the declined confirmation, and a missing selector are covered in
// smd_compep_errors_test.go.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSMDCompepGet_AllFormats verifies "get" (no args) formats output.
func TestSMDCompepGet_AllFormats(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ComponentEndpoints":[{"ID":"x3000c1s7b56n0"}]}`))
	}))
	defer srv.Close()

	for _, f := range []string{"json", "json-pretty", "yaml"} {
		res := runOchamiWithRuntime(t, "smd", "--ignore-config", "compep", "get", "--uri", srv.URL, "--token", "t", "-F", f)
		if res.err != nil {
			t.Fatalf("format %s: unexpected error: %v (exit %d)", f, res.err, res.exitCode)
		}
		assertFormattedOutput(t, f, res.stdout, "ID", "x3000c1s7b56n0")
	}
}

// TestSMDCompepGet_ByXnames verifies "get <xname>..." fetches per-endpoint and
// aggregates into a ComponentEndpoints array.
func TestSMDCompepGet_ByXnames(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ID":"x3000c1s7b56n0"}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "compep", "get", "--uri", srv.URL, "--token", "t",
		"x3000c1s7b56n0", "x3000c1s7b56n1")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "ComponentEndpoints") {
		t.Errorf("stdout = %q, want it to contain the ComponentEndpoints wrapper", res.stdout)
	}
}

// TestSMDCompepDelete_ByXnames verifies "delete --no-confirm <xname>..." issues a
// DELETE per endpoint.
func TestSMDCompepDelete_ByXnames(t *testing.T) {
	t.Parallel()

	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "compep", "delete", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "x3000c1s7b56n0", "x3000c1s7b56n1")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 2 {
		t.Errorf("DELETE count = %d, want 2", deletes)
	}
}

// TestSMDCompepDelete_ByData verifies IDs in a payload drive DELETE requests.
func TestSMDCompepDelete_ByData(t *testing.T) {
	t.Parallel()

	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "--ignore-config", "compep", "delete", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "-d", `[{"ID":"x3000c1s7b56n0"}]`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 1 {
		t.Errorf("DELETE count = %d, want 1", deletes)
	}
}

// TestSMDCompepDelete_AllConfirm verifies "delete --all" prompts and, on "y",
// issues a DELETE.
func TestSMDCompepDelete_AllConfirm(t *testing.T) {
	t.Parallel()

	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			gotMethod = r.Method
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInputAndRuntime(t, "y\n",
		"smd", "--ignore-config", "compep", "delete", "--uri", srv.URL, "--token", "t", "--all")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if !strings.Contains(res.stdout, "ALL COMPONENT ENDPOINTS") {
		t.Errorf("stdout = %q, want the all-endpoints confirmation prompt", res.stdout)
	}
}
