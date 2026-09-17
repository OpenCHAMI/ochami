// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// services_test.go provides representative end-to-end tests for the pcs,
// cloud_init, and metadata command groups: asserting request routing where the
// client uses direct endpoints, and asserting exit-code mapping for
// success/HTTP-failure cases.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// --- PCS ---

// TestPCSTransitionList verifies "pcs transition list" issues GET /transitions.
func TestPCSTransitionList(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"transitions":[]}`))
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "transition", "list", "--ignore-config", "--uri", srv.URL)

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodGet || gotPath != "/transitions" {
		t.Errorf("request = %s %s, want GET /transitions", gotMethod, gotPath)
	}
}

// TestPCSStatusShow verifies "pcs status show <xname>" issues GET /power-status
// and prints the first status entry.
func TestPCSStatusShow(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"status":[{"xname":"x3000c0s15b0","powerState":"on"}]}`))
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "status", "show", "--ignore-config", "--uri", srv.URL, "x3000c0s15b0")

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

// TestPCSStatusShowEmpty verifies that an empty status array resolves to
// CodeGeneric (the "no status found" case).
func TestPCSStatusShowEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":[]}`))
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "status", "show", "--ignore-config", "--uri", srv.URL, "x3000c0s15b0")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeGeneric {
		t.Errorf("exit code = %d, want %d (CodeGeneric)", res.exitCode, cli.CodeGeneric)
	}
}

// TestPCSTransitionStartInvalidOp verifies that an invalid operation argument
// is a usage error and no request is made.
func TestPCSTransitionStartInvalidOp(t *testing.T) {
	requestMade := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMade = true
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "transition", "start",
		"--ignore-config", "--uri", srv.URL, "--xname", "x0c0s0b0n0",
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

// --- cloud-init ---

// TestCloudInitDefaultsGet verifies "cloud-init defaults get" issues GET
// /admin/cluster-defaults.
func TestCloudInitDefaultsGet(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"cluster-name":"demo"}`))
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "defaults", "get", "--ignore-config", "--uri", srv.URL)

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/admin/cluster-defaults" {
		t.Errorf("path = %q, want /admin/cluster-defaults", gotPath)
	}
}

// TestCloudInitServiceStatusRunning verifies that "cloud-init service status"
// exits successfully when the /version endpoint responds OK.
func TestCloudInitServiceStatusRunning(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"version":"1.0.0"}`))
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "service", "status", "--ignore-config", "--uri", srv.URL)

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/version" {
		t.Errorf("path = %q, want /version", gotPath)
	}
	if !strings.Contains(res.stdout, "cloud-init is running") {
		t.Errorf("stdout = %q, want it to report running", res.stdout)
	}
}

// TestCloudInitServiceStatusNotRunning verifies that when the service is
// unreachable, "cloud-init service status" reports not running and resolves to
// a non-zero exit code.
func TestCloudInitServiceStatusNotRunning(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // connection refused

	res := runOchami(t, "cloud-init", "service", "status", "--ignore-config", "--uri", url)

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
	if !strings.Contains(res.stdout, "cloud-init is not running") {
		t.Errorf("stdout = %q, want it to report not running", res.stdout)
	}
}

// --- metadata ---

// TestMetadataGroupListHTTPError verifies that an unsuccessful HTTP response
// from the metadata service resolves to a non-success exit code. The metadata
// client wraps an upstream library, so we assert exit-code behavior rather than
// the exact request path.
func TestMetadataGroupListHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	res := runOchami(t, "metadata", "group", "list", "--ignore-config", "--uri", srv.URL, "--token", "faketoken")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode == cli.CodeSuccess {
		t.Errorf("exit code = %d, want a non-success code", res.exitCode)
	}
}

// TestMetadataGroupListSuccess verifies that "metadata group list" exits
// successfully when the service returns a valid list response.
func TestMetadataGroupListSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	res := runOchami(t, "metadata", "group", "list", "--ignore-config", "--uri", srv.URL, "--token", "faketoken")

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if res.exitCode != cli.CodeSuccess {
		t.Errorf("exit code = %d, want %d (CodeSuccess)", res.exitCode, cli.CodeSuccess)
	}
}

// TestPCSTransitionShow verifies "pcs transition show <id>" issues GET
// /transitions/<id>.
func TestPCSTransitionShow(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "transition", "show", "--ignore-config", "--uri", srv.URL, "abc-123")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodGet || gotPath != "/transitions/abc-123" {
		t.Errorf("request = %s %s, want GET /transitions/abc-123", gotMethod, gotPath)
	}
}

// TestPCSTransitionAbort verifies "pcs transition abort <id>" issues DELETE
// /transitions/<id>.
func TestPCSTransitionAbort(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "transition", "abort", "--ignore-config", "--uri", srv.URL, "abc-123")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete || gotPath != "/transitions/abc-123" {
		t.Errorf("request = %s %s, want DELETE /transitions/abc-123", gotMethod, gotPath)
	}
}

// TestPCSTransitionStart verifies "pcs transition start <op> --xname ..." issues
// POST /transitions.
func TestPCSTransitionStart(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "transition", "start", "--ignore-config", "--uri", srv.URL,
		"--xname", "x0c0s0b0n0", "on")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost || gotPath != "/transitions" {
		t.Errorf("request = %s %s, want POST /transitions", gotMethod, gotPath)
	}
}

// TestPCSTransitionMonitor verifies "pcs transition monitor <id>" polls
// /transitions/<id> and exits when the transition reports "completed".
func TestPCSTransitionMonitor(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		// Report completion immediately so the poll loop exits on the first
		// iteration without sleeping.
		_, _ = w.Write([]byte(`{"transitionStatus":"completed","taskCounts":{"total":1,"succeeded":1}}`))
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "transition", "monitor", "--ignore-config", "--uri", srv.URL, "abc-123")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/transitions/abc-123" {
		t.Errorf("path = %q, want /transitions/abc-123", gotPath)
	}
}
