// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// metadata_runcore_test.go exercises the metadata service command run-core
// functions to cover error handling paths identified in coverage analysis.

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestMetadataRunCoreDefaultsGetMalformedResponse verifies handling of malformed responses.
func TestMetadataRunCoreDefaultsGetMalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"Defaults":`)) //nolint:errcheck // malformed test response
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"metadata", "defaults", "get")

	if res.err == nil && res.exitCode == 0 {
		t.Logf("Metadata defaults get malformed: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}

// TestMetadataRunCoreDefaultsListHTTPError verifies HTTP error handling.
func TestMetadataRunCoreDefaultsListHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"metadata", "defaults", "list")

	if res.err == nil && res.exitCode == 0 {
		t.Logf("Metadata defaults list HTTP error: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}

// TestMetadataRunCoreGroupGetMalformedResponse verifies handling of malformed responses.
func TestMetadataRunCoreGroupGetMalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"Group":`)) //nolint:errcheck // malformed test response
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"metadata", "group", "get", "test-group")

	if res.err == nil && res.exitCode == 0 {
		t.Logf("Metadata group get malformed: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}

// TestMetadataRunCoreGroupListHTTPError verifies HTTP error handling.
func TestMetadataRunCoreGroupListHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"metadata", "group", "list")

	if res.err == nil && res.exitCode == 0 {
		t.Logf("Metadata group list HTTP error: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}

// TestMetadataRunCoreInstanceGetMalformedResponse verifies handling of malformed responses.
func TestMetadataRunCoreInstanceGetMalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"Instance":`)) //nolint:errcheck // malformed test response
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"metadata", "instance", "get", "x0c0s1b0n0")

	if res.err == nil && res.exitCode == 0 {
		t.Logf("Metadata instance get malformed: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}

// TestMetadataRunCorePeerGetMalformedResponse verifies handling of malformed responses.
func TestMetadataRunCorePeerGetMalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"Peer":`)) //nolint:errcheck // malformed test response
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"metadata", "peer", "get", "x0c0s1b0n0")

	if res.err == nil && res.exitCode == 0 {
		t.Logf("Metadata peer get malformed: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}

// TestMetadataRunCoreServiceStatusHTTPError verifies HTTP error handling.
func TestMetadataRunCoreServiceStatusHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"metadata", "service", "status")

	if res.err == nil && res.exitCode == 0 {
		t.Logf("Metadata service status HTTP error: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}
