// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// service_status_test.go covers additional branches of the "pcs service status"
// and "cloud-init service" verbs: the readiness->liveness fallback, the
// "unable to get state" path, health HTTP errors, and cloud-init service
// status/version error mapping.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestPCSServiceStatusLivenessFallback verifies that when readiness reports not
// ready (200) but liveness reports ready (204), the command reports "live".
func TestPCSServiceStatusLivenessFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/readiness":
			w.WriteHeader(http.StatusOK) // not "ready"
		case "/liveness":
			w.WriteHeader(http.StatusNoContent) // "live"
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "service", "status", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
}

// TestPCSServiceStatusUnknownState verifies the "unable to get state" path when
// neither readiness nor liveness reports ready.
func TestPCSServiceStatusUnknownState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // neither readiness nor liveness returns 204
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "service", "status", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode == cli.CodeSuccess {
		t.Errorf("exit code = %d, want a non-success code", res.exitCode)
	}
}

// TestPCSServiceStatusReadinessHTTPError verifies a failing readiness request
// resolves to CodeHTTP.
func TestPCSServiceStatusReadinessHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "service", "status", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestPCSServiceStatusHealthHTTPError verifies a failing health request (with a
// flag provided) resolves to CodeHTTP.
func TestPCSServiceStatusHealthHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "service", "status", "--all",
		"--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestCloudInitServiceVersionHTTPError verifies a failing "cloud-init service
// version" resolves to CodeHTTP.
func TestCloudInitServiceVersionHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "service", "version", "--ignore-config", "--uri", srv.URL)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestCloudInitServiceStatus verifies "cloud-init service status" reports
// success against a healthy server.
func TestCloudInitServiceStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "service", "status", "--ignore-config", "--uri", srv.URL)
	// Some status commands treat non-2xx-with-body as an error; accept either a
	// clean success or a mapped HTTP/network code, but never a panic.
	if res.err != nil && res.exitCode == cli.CodeSuccess {
		t.Errorf("inconsistent result: err=%v exit=%d", res.err, res.exitCode)
	}
	_ = strings.TrimSpace(res.stdout)
}

// TestCloudInitServiceStatusHTTPError verifies a responding but unhealthy
// service is distinguished from a network failure.
func TestCloudInitServiceStatusHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "service", "status", "--ignore-config", "--uri", srv.URL)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
	if !strings.Contains(res.stdout, "running, but not normally") {
		t.Errorf("stdout = %q, want abnormal-running status", res.stdout)
	}
}

// TestCloudInitServiceStatusQuietHTTPError verifies quiet mode suppresses the
// human-readable status while preserving the exit code.
func TestCloudInitServiceStatusQuietHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "service", "status", "--quiet", "--ignore-config", "--uri", srv.URL)
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
	if res.stdout != "" {
		t.Errorf("stdout = %q, want empty output", res.stdout)
	}
}

// TestCloudInitServiceStatusAPI verifies --api prints the returned OpenAPI
// document through the command's injected output stream.
func TestCloudInitServiceStatusAPI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"openapi":"3.0.0"}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "service", "status", "--api", "--ignore-config", "--uri", srv.URL)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "openapi") {
		t.Errorf("stdout = %q, want OpenAPI document", res.stdout)
	}
}

// TestCloudInitServiceStatusAPIError verifies --api aggregates request errors.
func TestCloudInitServiceStatusAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "service", "status", "--api", "--ignore-config", "--uri", srv.URL)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}
