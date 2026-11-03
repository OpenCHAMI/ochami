// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// pcs_test.go exercises the "pcs service" commands end-to-end against an
// httptest.Server. The status and transition commands are covered in
// pcs_status_test.go and pcs_transition_test.go.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

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

// TestPCSServiceStatus_HealthFalseFlags verifies that health flags passed as
// false are left out of the output while the ones passed as true are included.
func TestPCSServiceStatus_HealthFalseFlags(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.Write([]byte(`{"KvStore":"kv-ok","StateManager":"smd-ok","Vault":"vault-ok"}`))
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "pcs", "service", "status", "--storage=false", "--smd", "--vault=false",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "smd-ok") {
		t.Errorf("stdout = %q, want it to contain the smd status", res.stdout)
	}
	for _, unwanted := range []string{"kv-ok", "vault-ok"} {
		if strings.Contains(res.stdout, unwanted) {
			t.Errorf("stdout = %q, want it not to contain %q", res.stdout, unwanted)
		}
	}
}
