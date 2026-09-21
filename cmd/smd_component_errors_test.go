// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_component_errors_test.go covers HTTP/network/payload-failure paths for
// the "smd component" commands; see smd_component_test.go for the success
// paths these mirror.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestSMDComponentGet_AllHTTPError verifies that an unsuccessful HTTP response
// resolves to the CodeHTTP exit code.
func TestSMDComponentGet_AllHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "component", "get", "--ignore-config", "--uri", srv.URL)

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestSMDComponentGet_NetworkError verifies that a transport-level failure
// (server closed, connection refused) resolves to the CodeNetwork exit code.
func TestSMDComponentGet_NetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // close immediately so the connection is refused

	res := runOchamiWithRuntime(t, "smd", "component", "get", "--ignore-config", "--uri", url)

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
}

// TestSMDComponentAdd_BadPayload verifies that malformed -d payload data
// resolves to the CodePayload exit code before any request is made.
func TestSMDComponentAdd_BadPayload(t *testing.T) {
	requestMade := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMade = true
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "component", "add",
		"--ignore-config", "--uri", srv.URL,
		"--token", "faketoken",
		"-d", "{this is not valid json")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (CodePayload)", res.exitCode, cli.CodePayload)
	}
	if requestMade {
		t.Error("a request was made despite the payload being invalid")
	}
}

// TestSMDComponentAdd_MissingArgs verifies that invoking add without -d and
// without the required positional arguments is a usage error (CodeUsage).
func TestSMDComponentAdd_MissingArgs(t *testing.T) {
	res := runOchamiWithRuntime(t, "smd", "component", "add", "--ignore-config", "--uri", "http://127.0.0.1:0")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}

// TestSMDComponentDelete_PartialFailure verifies that a per-item unsuccessful
// HTTP response during multi-item deletion resolves to CodeHTTP (the
// "completed with errors" aggregate).
func TestSMDComponentDelete_PartialFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Fail every delete so the aggregate reports errors.
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "component", "delete",
		"--ignore-config", "--uri", srv.URL,
		"--token", "faketoken",
		"--no-confirm",
		"x3000c1s7b56n0")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}
