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

// TestPCSServiceStatus_UnknownState verifies the "unable to get state" path when
// neither readiness nor liveness reports ready.
func TestPCSServiceStatus_UnknownState(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // neither readiness nor liveness returns 204
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "pcs", "service", "status", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeGeneric {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeGeneric, cli.CodeName(cli.CodeGeneric))
	}
}

// TestPCSServiceStatus_ReadinessHTTPError verifies a failing readiness request
// resolves to CodeHTTP.
func TestPCSServiceStatus_ReadinessHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "pcs", "service", "status", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestPCSServiceStatus_HealthHTTPError verifies a failing health request (with a
// flag provided) resolves to CodeHTTP.
func TestPCSServiceStatus_HealthHTTPError(t *testing.T) {
	t.Parallel()

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
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
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
		w.Write([]byte(`{"Status":`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--uri", srv.URL, "--token", "t",
		"pcs", "service", "status")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeGeneric {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeGeneric, cli.CodeName(cli.CodeGeneric))
	}
}
