// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// pcs_transition_test.go exercises the "pcs transition" verbs (list, show,
// abort, start, monitor) end-to-end against an httptest.Server, including
// output-format variants. Error arms are covered in
// pcs_transition_errors_test.go.

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestPCSTransitionList_Formats verifies list output-format variants.
func TestPCSTransitionList_Formats(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"transitions":[{"transitionID":"t1"}]}`))
	}))
	defer srv.Close()

	for _, f := range []string{"json", "json-pretty", "yaml"} {
		res := runOchamiWithRuntime(t, "pcs", "--ignore-config", "transition", "list", "--uri", srv.URL, "-F", f)
		if res.err != nil {
			t.Fatalf("format %s: unexpected error: %v (exit %d)", f, res.err, res.exitCode)
		}
		assertFormattedOutput(t, f, res.stdout, "transitionID", "t1")
	}
}

// TestPCSTransitionList_Success verifies "pcs transition list" issues GET /transitions.
func TestPCSTransitionList_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Write([]byte(`{"transitions":[]}`))
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

// TestPCSTransitionShow_Success verifies "pcs transition show <id>" issues GET
// /transitions/<id>.
func TestPCSTransitionShow_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.Write([]byte(`{}`))
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
		w.Write([]byte(`{}`))
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
		w.Write([]byte(`{}`))
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
		w.Write([]byte(`{"transitionStatus":"completed","taskCounts":{"total":1,"succeeded":1}}`))
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
