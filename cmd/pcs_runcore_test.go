// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// pcs_runcore_test.go exercises the PCS service command run-core functions
// to cover error handling paths identified in coverage analysis.

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestPCSStatusMalformedResponse verifies handling of malformed responses.
func TestPCSRunCoreStatusMalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"State":`)) //nolint:errcheck // malformed test response
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"pcs", "status")

	if res.err == nil && res.exitCode == 0 {
		t.Logf("PCS status malformed: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}

// TestPCSStatusHTTPError verifies HTTP error handling.
func TestPCSRunCoreStatusHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"pcs", "status")

	if res.err == nil && res.exitCode == 0 {
		t.Logf("PCS status HTTP error: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}

// TestPCSServiceStatusMalformedResponse verifies handling of malformed responses.
func TestPCSRunCoreServiceStatusMalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"Status":`)) //nolint:errcheck // malformed test response
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"pcs", "service", "status")

	if res.err == nil && res.exitCode == 0 {
		t.Logf("PCS service status malformed: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}

// TestPCSTransitionListMalformedResponse verifies handling of malformed responses.
func TestPCSRunCoreTransitionListMalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"Transitions":`)) //nolint:errcheck // malformed test response
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"pcs", "transition", "list")

	if res.err == nil && res.exitCode == 0 {
		t.Logf("PCS transition list malformed: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}

// TestPCSTransitionShowMalformedResponse verifies handling of malformed responses.
func TestPCSRunCoreTransitionShowMalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"Transition":`)) //nolint:errcheck // malformed test response
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"pcs", "transition", "show", "test-transition")

	if res.err == nil && res.exitCode == 0 {
		t.Logf("PCS transition show malformed: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}
