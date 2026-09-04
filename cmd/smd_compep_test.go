// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_compep_test.go covers the branch families of the "smd compep" verbs (get,
// delete): the get-all vs get-by-xname arms, output-format variants, HTTP and
// network error mapping, per-item error aggregation, and the delete
// --all/args/-d selection with confirmation.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestSMDCompepGetAllFormats verifies "get" (no args) formats output.
func TestSMDCompepGetAllFormats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ComponentEndpoints":[{"ID":"x3000c1s7b56n0"}]}`))
	}))
	defer srv.Close()

	for _, f := range []string{"json", "json-pretty", "yaml"} {
		res := runOchami(t, "smd", "compep", "get", "--ignore-config", "--uri", srv.URL, "--token", "t", "-F", f)
		if res.err != nil {
			t.Fatalf("format %s: unexpected error: %v (exit %d)", f, res.err, res.exitCode)
		}
		if !strings.Contains(res.stdout, "x3000c1s7b56n0") {
			t.Errorf("format %s: stdout = %q, want it to contain the endpoint id", f, res.stdout)
		}
	}
}

// TestSMDCompepGetByXnames verifies "get <xname>..." fetches per-endpoint and
// aggregates into a ComponentEndpoints array.
func TestSMDCompepGetByXnames(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ID":"x3000c1s7b56n0"}`))
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "compep", "get", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"x3000c1s7b56n0", "x3000c1s7b56n1")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "ComponentEndpoints") {
		t.Errorf("stdout = %q, want it to contain the ComponentEndpoints wrapper", res.stdout)
	}
}

// TestSMDCompepGetAllHTTPError verifies an unsuccessful HTTP response on get-all
// resolves to CodeHTTP.
func TestSMDCompepGetAllHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "compep", "get", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestSMDCompepGetByXnameHTTPError verifies a failing per-xname get resolves to
// CodeHTTP via the aggregate.
func TestSMDCompepGetByXnameHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "compep", "get", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"x3000c1s7b56n0")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestSMDCompepGetNetworkError verifies a closed port resolves to CodeNetwork.
func TestSMDCompepGetNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close()

	res := runOchami(t, "smd", "compep", "get", "--ignore-config", "--uri", url, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
}

// TestSMDCompepDeleteByXnames verifies "delete --no-confirm <xname>..." issues a
// DELETE per endpoint.
func TestSMDCompepDeleteByXnames(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "compep", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "x3000c1s7b56n0", "x3000c1s7b56n1")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 2 {
		t.Errorf("DELETE count = %d, want 2", deletes)
	}
}

// TestSMDCompepDeleteAllConfirm verifies "delete --all" prompts and, on "y",
// issues a DELETE.
func TestSMDCompepDeleteAllConfirm(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			gotMethod = r.Method
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInput(t, "y\n",
		"smd", "compep", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t", "--all")
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

// TestSMDCompepDeleteAbort verifies answering "n" aborts without a request.
func TestSMDCompepDeleteAbort(t *testing.T) {
	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInput(t, "n\n",
		"smd", "compep", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t", "x3000c1s7b56n0")
	if res.err != nil {
		t.Fatalf("unexpected error on abort: %v (exit %d)", res.err, res.exitCode)
	}
	if deletes != 0 {
		t.Errorf("DELETE count = %d, want 0", deletes)
	}
}

// TestSMDCompepDeleteNoSelector verifies delete with neither -d, --all, nor args
// is a usage error.
func TestSMDCompepDeleteNoSelector(t *testing.T) {
	res := runOchami(t, "smd", "compep", "delete", "--ignore-config", "--uri", "http://127.0.0.1:1",
		"--token", "t", "--no-confirm")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}

// TestSMDCompepDeleteAllHTTPError verifies a failing "delete --all" resolves to
// CodeHTTP.
func TestSMDCompepDeleteAllHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "compep", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "--all")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}
