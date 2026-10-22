// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestPCSStatusList_InvalidPowerFilter verifies that an invalid --power-filter
// value is a usage error, reported before any request is sent.
func TestPCSStatusList_InvalidPowerFilter(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "pcs", "status", "list", "--ignore-config", "--uri", "http://127.0.0.1:0",
		"--token", "t", "--power-filter", "bogus")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
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
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
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
		w.Write([]byte(`{"State":`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--uri", srv.URL, "--token", "t",
		"pcs", "status", "list")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestPCSStatusShow_Empty verifies that an empty status array resolves to
// CodeGeneric (the "no status found" case).
func TestPCSStatusShow_Empty(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":[]}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "pcs", "status", "show", "--uri", srv.URL, "x3000c0s15b0")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeGeneric {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeGeneric, cli.CodeName(cli.CodeGeneric))
	}
}
