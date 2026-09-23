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
		w.Write([]byte(`{"status":[]}`))
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
		w.Write([]byte(`{"status":[]}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "pcs", "status", "list", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--xname", "x0c0s0b0n0", "--power-filter", "on", "--mgmt-filter", "available")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	for _, want := range []string{"xname=x0c0s0b0n0", "powerStateFilter=on", "managementStateFilter=available"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query = %q, want it to contain %q", gotQuery, want)
		}
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
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeSuccess, cli.CodeName(cli.CodeSuccess))
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
			w.Write([]byte(`{"KvStore":"ok","StateManager":"ok","Vault":"ok"}`))
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

// TestPCSServiceStatus_LivenessFallback verifies that "pcs service status"
// succeeds when PCS isn't ready but its liveness endpoint reports it alive.
func TestPCSServiceStatus_LivenessFallback(t *testing.T) {
	t.Parallel()

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

// TestPCSStatusShow_Success verifies "pcs status show <xname>" issues GET /power-status
// and prints the first status entry.
func TestPCSStatusShow_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Write([]byte(`{"status":[{"xname":"x3000c0s15b0","powerState":"on"}]}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "pcs", "status", "show", "--uri", srv.URL, "x3000c0s15b0")

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/power-status" {
		t.Errorf("path = %q, want /power-status", gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if !strings.Contains(res.stdout, "x3000c0s15b0") {
		t.Errorf("stdout = %q, want it to contain the xname", res.stdout)
	}
}
