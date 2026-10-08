// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
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
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "component", "get", "--ignore-config", "--uri", srv.URL)

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDComponentGet_NetworkError verifies that a transport-level failure
// (server closed, connection refused) resolves to the CodeNetwork exit code.
func TestSMDComponentGet_NetworkError(t *testing.T) {
	t.Parallel()

	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	res := runOchamiWithRuntime(t, "smd", "component", "get", "--ignore-config", "--uri", url)

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}

// TestSMDComponentAdd_BadPayload verifies that malformed -d payload data
// resolves to the CodePayload exit code before any request is made.
func TestSMDComponentAdd_BadPayload(t *testing.T) {
	t.Parallel()

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
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
	if requestMade {
		t.Error("a request was made despite the payload being invalid")
	}
}

// TestSMDComponentAdd_MissingArgs verifies that invoking add without -d and
// without the required positional arguments is a usage error (CodeUsage).
func TestSMDComponentAdd_MissingArgs(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "smd", "component", "add", "--ignore-config", "--uri", "http://127.0.0.1:0")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestSMDComponentDelete_PartialFailure verifies that a per-item unsuccessful
// HTTP response during multi-item deletion resolves to CodeHTTP (the
// "completed with errors" aggregate).
func TestSMDComponentDelete_PartialFailure(t *testing.T) {
	t.Parallel()

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
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDComponentDelete_Abort verifies answering "n" aborts without a request.
func TestSMDComponentDelete_Abort(t *testing.T) {
	t.Parallel()

	var deletes int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes++
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithInputAndRuntime(t, "n\n",
		"smd", "component", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t", "x3000c1s7b56n0")
	if res.exitCode != cli.CodeDeclined {
		t.Fatalf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, cli.CodeDeclined, cli.CodeName(cli.CodeDeclined))
	}
	if deletes != 0 {
		t.Errorf("DELETE count = %d, want 0", deletes)
	}
}

// TestSMDComponentGet_MalformedResponse verifies that "smd component get --nid"
// fails with CodePayload when the response can't be decoded.
func TestSMDComponentGet_MalformedResponse(t *testing.T) {
	t.Parallel()

	// Server returns malformed JSON
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Malformed JSON - missing closing brace
		w.Write([]byte(`{"Components":[{"ID":"x0c0s1b0n0"`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"smd", "component", "get", "--nid", "0")

	// Should fail due to malformed JSON
	if res.err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}

// TestSMDComponentGet_ByNIDHTTPError verifies that "smd component get --nid"
// fails with CodeHTTP for an HTTP 500 response.
func TestSMDComponentGet_ByNIDHTTPError(t *testing.T) {
	t.Parallel()

	// Server returns 500 Internal Server Error
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"smd", "component", "get", "--nid", "0")

	// Should fail with HTTP error
	if res.err == nil {
		t.Fatal("expected error for HTTP 500, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDComponentGet_ByNIDNetworkError verifies that a refused connection
// while getting a component by NID resolves to CodeNetwork.
func TestSMDComponentGet_ByNIDNetworkError(t *testing.T) {
	t.Parallel()

	// Nothing listens on port 1, so the connection is refused.
	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", "http://127.0.0.1:1", "--token", "t",
		"smd", "component", "get", "--nid", "0")

	if res.err == nil {
		t.Fatal("expected network error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
}
