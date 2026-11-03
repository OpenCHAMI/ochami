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

// TestPCSTransitionList_HTTPError verifies a failing list resolves to CodeHTTP.
func TestPCSTransitionList_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "pcs", "--ignore-config", "transition", "list", "--uri", srv.URL)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestPCSTransitionShow_HTTPError verifies a failing show resolves to CodeHTTP.
func TestPCSTransitionShow_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "pcs", "--ignore-config", "transition", "show", "--uri", srv.URL, "abcd-1234")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestPCSTransitionAbort_HTTPError verifies a failing abort resolves to CodeHTTP.
func TestPCSTransitionAbort_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "pcs", "--ignore-config", "transition", "abort", "--uri", srv.URL, "abcd-1234")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestPCSTransitionStart_HTTPError verifies a failing start resolves to CodeHTTP.
func TestPCSTransitionStart_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "pcs", "--ignore-config", "transition", "start", "--uri", srv.URL,
		"--xname", "x0c0s0b0n0", "on")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestPCSTransitionStart_InvalidOp verifies that an invalid operation argument
// is a usage error and no request is made.
func TestPCSTransitionStart_InvalidOp(t *testing.T) {
	t.Parallel()

	requestMade := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMade = true
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "pcs", "transition", "start",
		"--uri", srv.URL, "--xname", "x0c0s0b0n0",
		"bogus-operation")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
	if requestMade {
		t.Error("a request was made despite the operation being invalid")
	}
}

// TestPCSTransitionList_MalformedResponse verifies a malformed transition-list
// response resolves to CodePayload.
func TestPCSTransitionList_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Transitions":`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--uri", srv.URL, "--token", "t",
		"pcs", "transition", "list")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestPCSTransitionShow_MalformedResponse verifies a malformed transition-show
// response resolves to CodePayload.
func TestPCSTransitionShow_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Transition":`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--uri", srv.URL, "--token", "t",
		"pcs", "transition", "show", "test-transition")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestPCSTransitionStart_RequiresXname verifies --xname is enforced as a
// required flag (proving the MarkFlagRequired registration in start.go takes
// effect; the registration's own error return is unreachable in practice
// since "xname" is always registered immediately beforehand), and that
// Cobra's required-flag validation error resolves to CodeUsage via
// cli.WrapUsageErrors, which composes each command's PreRunE/PreRun to
// re-run and wrap Cobra's ValidateRequiredFlags/ValidateFlagGroups checks.
func TestPCSTransitionStart_RequiresXname(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "pcs", "transition", "start", "--ignore-config",
		"--uri", "http://127.0.0.1:1", "--token", "t", "on")
	if res.err == nil {
		t.Fatal("expected an error for missing required --xname, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}
