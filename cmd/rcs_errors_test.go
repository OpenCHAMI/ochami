// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// rcs_errors_test.go covers failure paths for the "rcs" command group:
// unsuccessful HTTP responses, and a console connection without a node ID, with
// its websocket upgrade rejected, or without a token; see rcs_test.go for the
// success paths these mirror.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestRCSConsoleList_HTTPError verifies an unsuccessful HTTP response resolves
// to CodeHTTP.
func TestRCSConsoleList_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "rcs", "console", "list", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestRCSConsoleConnect_MissingNodeID verifies that "rcs console connect"
// without a node ID is a usage error.
func TestRCSConsoleConnect_MissingNodeID(t *testing.T) {
	t.Parallel()

	// Try to connect without a node ID
	res := runOchamiWithRuntime(t, "--ignore-config", "rcs", "console", "connect")

	// Should fail with usage error
	if res.err == nil {
		t.Fatal("expected error for missing node ID, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestRCSConsoleConnect_UpgradeRejected verifies that a server which rejects
// the websocket upgrade (e.g. because it doesn't recognize the node ID)
// surfaces as a CodeNetwork error. There is no client-side node ID format
// validation in "rcs console connect" - the ID is passed straight through to
// the server, so this is the actual failure mode a bad ID produces.
func TestRCSConsoleConnect_UpgradeRejected(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "node not found", http.StatusNotFound)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--uri", srv.URL, "--token", "t",
		"rcs", "console", "connect", "invalid-node-id")

	if res.err == nil {
		t.Fatal("expected an error when the console upgrade is rejected, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}

// TestRCSConsoleConnect_RequiresToken verifies that an auth-enabled cluster
// rejects a missing token with CodeAuth before ever attempting the websocket
// dial. The --uri points at an address nothing listens on, so a CodeNetwork
// result (rather than CodeAuth) would mean the token check was skipped.
func TestRCSConsoleConnect_RequiresToken(t *testing.T) {
	t.Parallel()

	cfg := writeTempConfig(t, `clusters:
  - name: secure
    cluster:
      uri: http://127.0.0.1:1
      enable-auth: true
`)

	res := runOchamiWithRuntime(t, "--config", cfg, "--cluster", "secure",
		"rcs", "console", "connect", "x0c0s1b0n0")

	if res.err == nil {
		t.Fatal("expected an error for a missing token on an auth-enabled cluster, got nil")
	}
	if res.exitCode != cli.CodeAuth {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeAuth, cli.CodeName(cli.CodeAuth))
	}
}
