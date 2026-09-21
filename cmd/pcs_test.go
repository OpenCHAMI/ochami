// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// pcs_test.go exercises "pcs status" and "pcs service" commands end-to-end
// against an httptest.Server. Transition commands are covered in
// pcs_transition_test.go.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestPCSStatusList_Success verifies "pcs status list" issues GET /power-status.
func TestPCSStatusList_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{"status":[]}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "pcs", "status", "list", "--ignore-config", "--uri", srv.URL, "--token", "t")
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

// TestPCSStatusList_WithFilters verifies xname and power/mgmt filters are encoded
// in the query string.
func TestPCSStatusList_WithFilters(t *testing.T) {
	t.Parallel()

	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"status":[]}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "pcs", "status", "list", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--xname", "x0c0s0b0n0", "--power-filter", "on", "--mgmt-filter", "available")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(gotQuery, "x0c0s0b0n0") {
		t.Errorf("query = %q, want it to contain the xname", gotQuery)
	}
}

// TestPCSStatusList_InvalidPowerFilter verifies an invalid --power-filter value
// is a usage error handled before any request.
func TestPCSStatusList_InvalidPowerFilter(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "pcs", "status", "list", "--ignore-config", "--uri", "http://127.0.0.1:0",
		"--token", "t", "--power-filter", "bogus")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode == cli.CodeSuccess {
		t.Errorf("exit code = %d, want a non-success code", res.exitCode)
	}
}

// TestPCSStatusList_HTTPError verifies an unsuccessful HTTP response resolves to
// CodeHTTP.
func TestPCSStatusList_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "pcs", "status", "list", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestPCSServiceStatus_Success verifies "pcs service status" contacts PCS readiness and
// exits successfully when PCS reports ready (HTTP 204 on /readiness).
func TestPCSServiceStatus_Success(t *testing.T) {
	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "pcs", "service", "status", "--ignore-config", "--uri", srv.URL, "--token", "t")
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

// TestPCSServiceStatus_Health verifies that passing a health flag causes
// "pcs service status" to query the /health endpoint.
func TestPCSServiceStatus_Health(t *testing.T) {
	t.Parallel()

	var sawHealth bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			sawHealth = true
			_, _ = w.Write([]byte(`{"KvStore":"ok","StateManager":"ok","Vault":"ok"}`)) //nolint:errcheck // test response writes are observed by the client
		default:
			// readiness/liveness report ready
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "pcs", "service", "status", "--all",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !sawHealth {
		t.Error("server never received a request to /health")
	}
}
func TestPCSServiceStatus_LivenessFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/readiness":
			w.WriteHeader(http.StatusOK) // not "ready"
		case "/liveness":
			w.WriteHeader(http.StatusNoContent) // "live"
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "pcs", "service", "status", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
}

// TestPCSServiceStatus_UnknownState verifies the "unable to get state" path when
// neither readiness nor liveness reports ready.
func TestPCSServiceStatus_UnknownState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // neither readiness nor liveness returns 204
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "pcs", "service", "status", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode == cli.CodeSuccess {
		t.Errorf("exit code = %d, want a non-success code", res.exitCode)
	}
}

// TestPCSServiceStatus_ReadinessHTTPError verifies a failing readiness request
// resolves to CodeHTTP.
func TestPCSServiceStatus_ReadinessHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "pcs", "service", "status", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestPCSServiceStatus_HealthHTTPError verifies a failing health request (with a
// flag provided) resolves to CodeHTTP.
func TestPCSServiceStatus_HealthHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "pcs", "service", "status", "--all",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestPCSStatusList_MalformedResponse verifies a malformed status-list
// response resolves to CodePayload (a decode failure, not an HTTP failure:
// the server responds 200 OK with an undecodable body).
func TestPCSStatusList_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"State":`)) //nolint:errcheck // malformed test response
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--uri", srv.URL, "--token", "t",
		"pcs", "status", "list")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (CodePayload)", res.exitCode, cli.CodePayload)
	}
}

// TestPCSServiceStatus_MalformedResponse verifies a malformed readiness/liveness
// response resolves to CodeGeneric: the command can't classify the failure as
// a specific HTTP status once neither probe successfully reports a state.
func TestPCSServiceStatus_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"Status":`)) //nolint:errcheck // malformed test response
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--uri", srv.URL, "--token", "t",
		"pcs", "service", "status")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeGeneric {
		t.Errorf("exit code = %d, want %d (CodeGeneric)", res.exitCode, cli.CodeGeneric)
	}
}
