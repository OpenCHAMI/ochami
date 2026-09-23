// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// rcs_errors_test.go covers HTTP-failure paths for the "rcs" command group;
// see rcs_test.go for the success paths these mirror.

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRCSConsoleList_HTTPError verifies an unsuccessful HTTP response resolves to
// a non-zero exit code.
func TestRCSConsoleList_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "rcs", "console", "list", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode == 0 {
		t.Errorf("exit code = %d, want non-zero", res.exitCode)
	}
}
