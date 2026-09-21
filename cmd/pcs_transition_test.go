// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// pcs_transition_test.go covers additional branch families of the "pcs
// transition" verbs (list, show, abort, start): output-format variants, HTTP
// error mapping, and the missing-required-flag usage error on start.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestPCSTransitionList_Formats verifies list output-format variants.
func TestPCSTransitionList_Formats(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"transitions":[]}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	for _, f := range []string{"json", "json-pretty", "yaml"} {
		res := runOchamiWithRuntime(t, "pcs", "--ignore-config", "transition", "list", "--uri", srv.URL, "-F", f)
		if res.err != nil {
			t.Fatalf("format %s: unexpected error: %v (exit %d)", f, res.err, res.exitCode)
		}
	}
}

// TestPCSTransitionList_HTTPError verifies a failing list resolves to CodeHTTP.
func TestPCSTransitionList_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	t.Parallel()
	res := runOchamiWithRuntime(t, "pcs", "--ignore-config", "transition", "list", "--uri", srv.URL)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestPCSTransitionShow_HTTPError verifies a failing show resolves to CodeHTTP.
func TestPCSTransitionShow_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	t.Parallel()
	res := runOchamiWithRuntime(t, "pcs", "--ignore-config", "transition", "show", "--uri", srv.URL, "abcd-1234")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestPCSTransitionAbort_HTTPError verifies a failing abort resolves to CodeHTTP.
func TestPCSTransitionAbort_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	t.Parallel()
	res := runOchamiWithRuntime(t, "pcs", "--ignore-config", "transition", "abort", "--uri", srv.URL, "abcd-1234")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestPCSTransitionStart_HTTPError verifies a failing start resolves to CodeHTTP.
func TestPCSTransitionStart_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	t.Parallel()
	res := runOchamiWithRuntime(t, "pcs", "--ignore-config", "transition", "start", "--uri", srv.URL,
		"--xname", "x0c0s0b0n0", "on")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestPCSTransitionStart_MissingXname verifies "start <op>" without the required
// --xname flag fails (non-success exit).
func TestPCSTransitionStart_MissingXname(t *testing.T) {

	t.Parallel()
	res := runOchamiWithRuntime(t, "pcs", "--ignore-config", "transition", "start", "--uri", "http://127.0.0.1:1", "on")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode == cli.CodeSuccess {
		t.Errorf("exit code = %d, want a non-success code", res.exitCode)
	}
}

// TestPCSTransitionList_Success verifies "pcs transition list" issues GET /transitions.
func TestPCSTransitionList_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"transitions":[]}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "pcs", "transition", "list", "--uri", srv.URL)

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodGet || gotPath != "/transitions" {
		t.Errorf("request = %s %s, want GET /transitions", gotMethod, gotPath)
	}
}

// TestPCSStatusShow_Success verifies "pcs status show <xname>" issues GET /power-status
// and prints the first status entry.
func TestPCSStatusShow_Success(t *testing.T) {
	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"status":[{"xname":"x3000c0s15b0","powerState":"on"}]}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "pcs", "status", "show", "--uri", srv.URL, "x3000c0s15b0")

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/power-status" {
		t.Errorf("path = %q, want /power-status", gotPath)
	}
	if !strings.Contains(res.stdout, "x3000c0s15b0") {
		t.Errorf("stdout = %q, want it to contain the xname", res.stdout)
	}
}

// TestPCSStatusShow_Empty verifies that an empty status array resolves to
// CodeGeneric (the "no status found" case).
func TestPCSStatusShow_Empty(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":[]}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "pcs", "status", "show", "--uri", srv.URL, "x3000c0s15b0")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeGeneric {
		t.Errorf("exit code = %d, want %d (CodeGeneric)", res.exitCode, cli.CodeGeneric)
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
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
	if requestMade {
		t.Error("a request was made despite the operation being invalid")
	}
}

// TestPCSTransitionShow_Success verifies "pcs transition show <id>" issues GET
// /transitions/<id>.
func TestPCSTransitionShow_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "pcs", "transition", "show", "--uri", srv.URL, "abc-123")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodGet || gotPath != "/transitions/abc-123" {
		t.Errorf("request = %s %s, want GET /transitions/abc-123", gotMethod, gotPath)
	}
}

// TestPCSTransitionAbort_Success verifies "pcs transition abort <id>" issues DELETE
// /transitions/<id>.
func TestPCSTransitionAbort_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "pcs", "transition", "abort", "--uri", srv.URL, "abc-123")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete || gotPath != "/transitions/abc-123" {
		t.Errorf("request = %s %s, want DELETE /transitions/abc-123", gotMethod, gotPath)
	}
}

// TestPCSTransitionStart_Success verifies "pcs transition start <op> --xname ..." issues
// POST /transitions.
func TestPCSTransitionStart_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "pcs", "transition", "start", "--uri", srv.URL,
		"--xname", "x0c0s0b0n0", "on")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost || gotPath != "/transitions" {
		t.Errorf("request = %s %s, want POST /transitions", gotMethod, gotPath)
	}
}

// TestPCSTransitionMonitor_Success verifies "pcs transition monitor <id>" polls
// /transitions/<id> and exits when the transition reports "completed".
func TestPCSTransitionMonitor_Success(t *testing.T) {
	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		// Report completion immediately so the poll loop exits on the first
		// iteration without sleeping.
		_, _ = w.Write([]byte(`{"transitionStatus":"completed","taskCounts":{"total":1,"succeeded":1}}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "pcs", "transition", "monitor", "--uri", srv.URL, "abc-123")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/transitions/abc-123" {
		t.Errorf("path = %q, want /transitions/abc-123", gotPath)
	}
}

// TestPCSTransitionList_MalformedResponse verifies a malformed transition-list
// response resolves to CodePayload.
func TestPCSTransitionList_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"Transitions":`)) //nolint:errcheck // malformed test response
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--uri", srv.URL, "--token", "t",
		"pcs", "transition", "list")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (CodePayload)", res.exitCode, cli.CodePayload)
	}
}

// TestPCSTransitionShow_MalformedResponse verifies a malformed transition-show
// response resolves to CodePayload.
func TestPCSTransitionShow_MalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"Transition":`)) //nolint:errcheck // malformed test response
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--uri", srv.URL, "--token", "t",
		"pcs", "transition", "show", "test-transition")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (CodePayload)", res.exitCode, cli.CodePayload)
	}
}
