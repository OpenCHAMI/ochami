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
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestPCSTransitionListFormats verifies list output-format variants.
func TestPCSTransitionListFormats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"transitions":[]}`))
	}))
	defer srv.Close()

	for _, f := range []string{"json", "json-pretty", "yaml"} {
		res := runOchami(t, "pcs", "transition", "list", "--ignore-config", "--uri", srv.URL, "-F", f)
		if res.err != nil {
			t.Fatalf("format %s: unexpected error: %v (exit %d)", f, res.err, res.exitCode)
		}
	}
}

// TestPCSTransitionListHTTPError verifies a failing list resolves to CodeHTTP.
func TestPCSTransitionListHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "transition", "list", "--ignore-config", "--uri", srv.URL)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestPCSTransitionShowHTTPError verifies a failing show resolves to CodeHTTP.
func TestPCSTransitionShowHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "transition", "show", "--ignore-config", "--uri", srv.URL, "abcd-1234")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestPCSTransitionAbortHTTPError verifies a failing abort resolves to CodeHTTP.
func TestPCSTransitionAbortHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "transition", "abort", "--ignore-config", "--uri", srv.URL, "abcd-1234")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestPCSTransitionStartHTTPError verifies a failing start resolves to CodeHTTP.
func TestPCSTransitionStartHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "transition", "start", "--ignore-config", "--uri", srv.URL,
		"--xname", "x0c0s0b0n0", "on")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestPCSTransitionStartMissingXname verifies "start <op>" without the required
// --xname flag fails (non-success exit).
func TestPCSTransitionStartMissingXname(t *testing.T) {
	res := runOchami(t, "pcs", "transition", "start", "--ignore-config", "--uri", "http://127.0.0.1:1", "on")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode == cli.CodeSuccess {
		t.Errorf("exit code = %d, want a non-success code", res.exitCode)
	}
}
