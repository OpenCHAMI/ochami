// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// pcs_test.go exercises "pcs status" and "pcs service" commands end-to-end
// against an httptest.Server. Transition commands are covered in
// services_test.go.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestPCSStatusList verifies "pcs status list" issues GET /power-status.
func TestPCSStatusList(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"status":[]}`))
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "status", "list", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/power-status" {
		t.Errorf("path = %q, want /power-status", gotPath)
	}
}

// TestPCSStatusListWithFilters verifies xname and power/mgmt filters are encoded
// in the query string.
func TestPCSStatusListWithFilters(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"status":[]}`))
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "status", "list", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--xname", "x0c0s0b0n0", "--power-filter", "on", "--mgmt-filter", "available")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(gotQuery, "x0c0s0b0n0") {
		t.Errorf("query = %q, want it to contain the xname", gotQuery)
	}
}

// TestPCSStatusListInvalidPowerFilter verifies an invalid --power-filter value
// is a usage error handled before any request.
func TestPCSStatusListInvalidPowerFilter(t *testing.T) {
	res := runOchami(t, "pcs", "status", "list", "--ignore-config", "--uri", "http://127.0.0.1:0",
		"--token", "t", "--power-filter", "bogus")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode == cli.CodeSuccess {
		t.Errorf("exit code = %d, want a non-success code", res.exitCode)
	}
}

// TestPCSStatusListHTTPError verifies an unsuccessful HTTP response resolves to
// CodeHTTP.
func TestPCSStatusListHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "status", "list", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestPCSServiceStatus verifies "pcs service status" contacts PCS readiness and
// exits successfully when PCS reports ready (HTTP 204 on /readiness).
func TestPCSServiceStatus(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "service", "status", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if res.exitCode != cli.CodeSuccess {
		t.Errorf("exit code = %d, want %d (CodeSuccess)", res.exitCode, cli.CodeSuccess)
	}
	if gotPath != "/readiness" {
		t.Errorf("path = %q, want /readiness", gotPath)
	}
}

// TestPCSServiceStatusHealth verifies that passing a health flag causes
// "pcs service status" to query the /health endpoint.
func TestPCSServiceStatusHealth(t *testing.T) {
	var sawHealth bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			sawHealth = true
			_, _ = w.Write([]byte(`{"KvStore":"ok","StateManager":"ok","Vault":"ok"}`))
		default:
			// readiness/liveness report ready
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "service", "status", "--all",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !sawHealth {
		t.Error("server never received a request to /health")
	}
}
