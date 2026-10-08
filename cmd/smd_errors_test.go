// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_errors_test.go covers HTTP- and transport-failure paths for the "smd"
// command group; see smd_test.go for the success paths these mirror.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestSMDDeprecatedStatus_HTTPError verifies that an unsuccessful HTTP response from the
// SMD service status endpoint resolves to CodeHTTP.
func TestSMDDeprecatedStatus_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "service", "status", "--ignore-config", "--uri", srv.URL)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestSMDGroupGet_TruncatedResponseBody verifies that a truncated HTTP response
// body (a declared Content-Length the server doesn't deliver) is reported as
// a network error, not silently swallowed.
func TestSMDGroupGet_TruncatedResponseBody(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Declare a body longer than what's written, then close the
		// connection without fulfilling Content-Length.
		w.Header().Set("Content-Length", "100")
		w.Write([]byte("partial"))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "smd", "group", "get", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error for a truncated response body, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
	if !strings.Contains(res.err.Error(), "unexpected EOF") {
		t.Errorf("error = %q, want it to report the truncated read", res.err.Error())
	}
}
