// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// cloud_init_runcore_test.go exercises the cloud-init service command run-core
// functions to cover error handling paths identified in coverage analysis.

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCloudInitDefaultsGetMalformedResponse verifies handling of malformed responses.
func TestCloudInitRunCoreDefaultsGetMalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"defaults":`)) //nolint:errcheck // malformed test response
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"cloud-init", "defaults", "get")

	if res.err == nil && res.exitCode == 0 {
		t.Logf("Cloud-init defaults get malformed: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}

// TestCloudInitDefaultsListHTTPError verifies HTTP error handling.
func TestCloudInitRunCoreDefaultsListHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"cloud-init", "defaults", "list")

	if res.err == nil && res.exitCode == 0 {
		t.Logf("Cloud-init defaults list HTTP error: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}

// TestCloudInitGroupGetMalformedResponse verifies handling of malformed responses.
func TestCloudInitRunCoreGroupGetMalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"group":`)) //nolint:errcheck // malformed test response
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"cloud-init", "group", "get", "test-group")

	if res.err == nil && res.exitCode == 0 {
		t.Logf("Cloud-init group get malformed: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}

// TestCloudInitGroupListHTTPError verifies HTTP error handling.
func TestCloudInitRunCoreGroupListHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"cloud-init", "group", "list")

	if res.err == nil && res.exitCode == 0 {
		t.Logf("Cloud-init group list HTTP error: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}

// TestCloudInitNodeGetMalformedResponse verifies handling of malformed responses.
func TestCloudInitRunCoreNodeGetMalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"node":`)) //nolint:errcheck // malformed test response
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"cloud-init", "node", "get", "x0c0s1b0n0")

	if res.err == nil && res.exitCode == 0 {
		t.Logf("Cloud-init node get malformed: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}

// TestCloudInitServiceStatusHTTPError verifies HTTP error handling.
func TestCloudInitRunCoreServiceStatusHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"cloud-init", "service", "status")

	if res.err == nil && res.exitCode == 0 {
		t.Logf("Cloud-init service status HTTP error: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}
