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

// TestBSSBootParamsSet_HTTPError verifies an unsuccessful HTTP response resolves
// to CodeHTTP.
func TestBSSBootParamsSet_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "bss", "boot", "params", "set",
		"--ignore-config", "--uri", srv.URL, "--token", "faketoken",
		"--mac", "de:ad:be:ef:00:00", "--kernel", "https://example.com/vmlinuz")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}
