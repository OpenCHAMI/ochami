// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestCloudInitNodeGetData_HTTPError verifies an unsuccessful HTTP response from
// a node get resolves to CodeHTTP.
func TestCloudInitNodeGetData_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "--ignore-config", "node", "get", "meta-data", "x3000c0s0b0n0",
		"--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestCloudInitNodeGet_MetadataHTTPError verifies a failing meta-data fetch
// resolves to CodeHTTP via the aggregate.
func TestCloudInitNodeGet_MetadataHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "node", "get", "meta-data", "--ignore-config",
		"--uri", srv.URL, "--token", "t", "x0c0s0b0n0")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestCloudInitNodeGet_UserdataHTTPError verifies a failing user-data fetch
// resolves to CodeHTTP.
func TestCloudInitNodeGet_UserdataHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "node", "get", "user-data", "--ignore-config",
		"--uri", srv.URL, "--token", "t", "x0c0s0b0n0")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestCloudInitNodeGet_GroupHTTPError verifies a failing node-group fetch
// resolves to CodeHTTP via the aggregate.
func TestCloudInitNodeGet_GroupHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "node", "get", "group", "--ignore-config",
		"--uri", srv.URL, "--token", "t", "x0c0s0b0n0", "compute")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestCloudInitNodeSet_HTTPError verifies a failing "node set" resolves to
// CodeHTTP via the aggregate.
func TestCloudInitNodeSet_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "node", "set", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"-d", `[{"id":"x0c0s0b0n0"}]`)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestCloudInitNodeSet_MalformedPayload verifies malformed inline payload
// resolves to CodePayload.
func TestCloudInitNodeSet_MalformedPayload(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "node", "set", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"-d", `not json`)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}
