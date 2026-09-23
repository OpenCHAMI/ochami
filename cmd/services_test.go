// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// services_test.go provides representative end-to-end tests for the pcs,
// cloud_init, and metadata command groups: asserting request routing where the
// client uses direct endpoints, and asserting exit-code mapping for
// success/HTTP-failure cases. HTTP-failure and rejection-path cases are
// covered in services_errors_test.go.

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// --- PCS ---

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

// --- cloud-init ---

// TestCloudInitDefaultsGet verifies "cloud-init defaults get" issues GET
// /admin/cluster-defaults.
func TestCloudInitDefaultsGet(t *testing.T) {
	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"cluster-name":"demo"}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "defaults", "get", "--uri", srv.URL)

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/admin/cluster-defaults" {
		t.Errorf("path = %q, want /admin/cluster-defaults", gotPath)
	}
}

// TestCloudInitServiceStatus_Running verifies that "cloud-init service status"
// exits successfully when the /version endpoint responds OK.
func TestCloudInitServiceStatus_Running(t *testing.T) {
	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"version":"1.0.0"}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "service", "status", "--uri", srv.URL)

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

// --- metadata ---

// TestMetadataGroupList_Success verifies that "metadata group list" exits
// successfully when the service returns a valid list response.
func TestMetadataGroupList_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "metadata", "group", "list", "--uri", srv.URL, "--token", "faketoken")

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if res.exitCode != cli.CodeSuccess {
		t.Errorf("exit code = %d, want %d (CodeSuccess)", res.exitCode, cli.CodeSuccess)
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

// TestRemainingServicePaths covers the remaining per-service version/status
// routes not already exercised by their own family test files.
func TestRemainingServicePaths(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPath string
		body     string
	}{
		{"bss version", []string{"bss", "service", "version"}, "/service/version", `{"version":"1.0"}`},
		{"cloud-init version", []string{"cloud-init", "service", "version"}, "/version", `{"version":"1.0"}`},
		{"metadata status", []string{"metadata", "service", "status"}, "/health", `{}`},
		{"rcs status", []string{"rcs", "service", "status", "--token", "t"}, "/health", `{"status":"ok"}`},
		{"deprecated smd status", []string{"smd", "status"}, "/service/ready", `{}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, tc.body) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			args := append(append([]string{}, tc.args...), "--ignore-config", "--uri", srv.URL)
			res := runOchamiWithRuntime(t, args...)
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if gotPath != tc.wantPath {
				t.Errorf("path = %q, want %q", gotPath, tc.wantPath)
			}
		})
	}
}
