// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// bss_errors_test.go covers HTTP-failure paths for the "bss" command group;
// see bss_test.go for the success paths these mirror.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestBSSBootParamsAdd_MissingSelectors verifies that add without -d and without
// any of --xname/--nid/--mac is a usage error.
func TestBSSBootParamsAdd_MissingSelectors(t *testing.T) {
	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "add",
		"--ignore-config", "--uri", "http://127.0.0.1:0", "--token", "faketoken",
		"--kernel", "https://example.com/vmlinuz")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}

// TestBSSServiceStatus_HTTPError verifies an unsuccessful HTTP response from the
// status endpoint resolves to CodeHTTP.
func TestBSSServiceStatus_HTTPError(t *testing.T) {
	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "service", "status", "--ignore-config", "--uri", srv.URL)

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}
