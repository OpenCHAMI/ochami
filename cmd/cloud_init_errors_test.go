// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestCloudInitServiceVersion_HTTPError verifies that "cloud-init service
// version" resolves an unsuccessful HTTP response to CodeHTTP.
func TestCloudInitServiceVersion_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "service", "version", "--ignore-config", "--uri", srv.URL)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestCloudInitServiceStatus_HTTPError verifies that "cloud-init service
// status" reports a service that responds with HTTP 500 as running abnormally
// and fails with CodeHTTP.
func TestCloudInitServiceStatus_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "service", "status", "--ignore-config", "--uri", srv.URL)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
	if !strings.Contains(res.stdout, "running, but not normally") {
		t.Errorf("stdout = %q, want abnormal-running status", res.stdout)
	}
}

// TestCloudInitServiceStatus_QuietHTTPError verifies quiet mode suppresses the
// human-readable status while preserving the exit code.
func TestCloudInitServiceStatus_QuietHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "service", "status", "--quiet", "--ignore-config", "--uri", srv.URL)
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
	if res.stdout != "" {
		t.Errorf("stdout = %q, want empty output", res.stdout)
	}
}

// TestCloudInitServiceStatus_APIError verifies that "cloud-init service status
// --api" resolves an unsuccessful HTTP response to CodeHTTP.
func TestCloudInitServiceStatus_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "service", "status", "--api", "--ignore-config", "--uri", srv.URL)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestCloudInitServiceStatus_NotRunning verifies that when the service is
// unreachable, "cloud-init service status" reports not running and resolves to
// CodeNetwork.
func TestCloudInitServiceStatus_NotRunning(t *testing.T) {
	t.Parallel()

	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "service", "status", "--uri", url)

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
	if !strings.Contains(res.stdout, "cloud-init is not running") {
		t.Errorf("stdout = %q, want it to report not running", res.stdout)
	}
}
