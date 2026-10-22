// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// cloud_init_test.go exercises the "cloud-init service" commands end-to-end
// against an httptest.Server, asserting outbound request method/path and the
// resolved exit code. The group, node, and defaults commands are covered in
// their own cloud_init_<command>_test.go files.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCloudInitServiceStatus_API verifies --api prints the returned OpenAPI
// document through the command's injected output stream.
func TestCloudInitServiceStatus_API(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"openapi":"3.0.0"}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "service", "status", "--api", "--ignore-config", "--uri", srv.URL)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "openapi") {
		t.Errorf("stdout = %q, want OpenAPI document", res.stdout)
	}
}

// TestCloudInitServiceStatus_Running verifies that "cloud-init service status"
// exits successfully when the /version endpoint responds OK.
func TestCloudInitServiceStatus_Running(t *testing.T) {
	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`{"version":"1.0.0"}`))
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

// TestCloudInitServiceStatus_QuietFalse verifies that --quiet=false prints the
// running status, as if --quiet were omitted.
func TestCloudInitServiceStatus_QuietFalse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"version":"1.0.0"}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "service", "status", "--quiet=false", "--uri", srv.URL)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "cloud-init is running") {
		t.Errorf("stdout = %q, want it to report running", res.stdout)
	}
}
