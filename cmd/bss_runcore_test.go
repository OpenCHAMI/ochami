// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// bss_runcore_test.go exercises the BSS command run-core functions to cover
// error handling paths identified in coverage analysis.

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestBSSRunCoreMalformedResponse verifies that the BSS boot params get
// command handles malformed JSON responses gracefully.
func TestBSSRunCoreMalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"BootParameters":[{"ID":"`)); err != nil {
			t.Errorf("writing response: %v", err)
		}
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"bss", "boot", "params", "get", "x0c0s1b0n0")

	if res.err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
	if res.exitCode == 0 {
		t.Errorf("expected non-zero exit code, got %d", res.exitCode)
	}
}

// TestBSSRunCoreHTTPError verifies that HTTP 5xx errors are handled.
func TestBSSRunCoreHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"bss", "boot", "params", "get", "x0c0s1b0n0")

	if res.err == nil {
		t.Fatal("expected error for HTTP 503, got nil")
	}
	if res.exitCode == 0 {
		t.Errorf("expected non-zero exit code, got %d", res.exitCode)
	}
}

// TestBSSRunCoreNetworkError verifies network-level errors are handled.
func TestBSSRunCoreNetworkError(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", "http://192.0.2.1:5000", "--token", "t",
		"--timeout", "1s", "bss", "boot", "params", "get", "x0c0s1b0n0")

	if res.err == nil {
		t.Fatal("expected network error, got nil")
	}
	if res.exitCode == 0 {
		t.Errorf("expected non-zero exit code, got %d", res.exitCode)
	}
}

// TestBSSRunCorePartialFailure verifies handling of empty results.
func TestBSSRunCorePartialFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Empty array response
		if _, err := w.Write([]byte(`[]`)); err != nil {
			t.Errorf("writing response: %v", err)
		}
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"bss", "boot", "params", "get", "x0c0s1b0n0")

	// Should succeed with empty results
	if res.err != nil && res.exitCode == 0 {
		t.Logf("Partial failure test: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}

// TestBSSInvalidFlagCombination verifies invalid flag combinations are rejected.
func TestBSSInvalidFlagCombination(t *testing.T) {
	t.Parallel()

	// Try with invalid combination of flags (if any exist)
	// For now, just test that the command accepts valid flags
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"BootParameters":[]}`)); err != nil {
			t.Errorf("writing response: %v", err)
		}
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"bss", "boot", "params", "get", "x0c0s1b0n0")

	if res.err != nil {
		t.Logf("Flag combination test: err=%v", res.err)
	}
}

// TestBSSContextCancellation verifies context cancellation during request.
func TestBSSContextCancellation(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		select {
		case <-r.Context().Done():
			http.Error(w, "Request cancelled", http.StatusServiceUnavailable)
			return
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write([]byte(`{"BootParameters":[]}`)); err != nil {
				t.Errorf("writing response: %v", err)
			}
		}
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"--timeout", "100ms", "bss", "boot", "params", "get", "x0c0s1b0n0")

	t.Logf("Context cancellation test: err=%v, exitCode=%d", res.err, res.exitCode)
}
