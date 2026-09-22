// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_errors_test.go covers HTTP-failure paths for the "smd" command group;
// see smd_test.go for the success paths these mirror.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestSMDDeprecatedStatus_HTTPError verifies that an unsuccessful HTTP response from the
// SMD service status endpoint resolves to CodeHTTP.
func TestSMDDeprecatedStatus_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	res := runOchami(t, "smd", "service", "status", "--ignore-config", "--uri", srv.URL)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}
