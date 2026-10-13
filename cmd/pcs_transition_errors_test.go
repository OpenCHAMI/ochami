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

// TestPCSTransitionStart_InvalidOp verifies that an invalid operation argument
// is a usage error and no request is made.
func TestPCSTransitionStart_InvalidOp(t *testing.T) {
	requestMade := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMade = true
	}))
	defer srv.Close()

	res := runOchami(t, "pcs", "transition", "start",
		"--ignore-config", "--uri", srv.URL, "--xname", "x0c0s0b0n0",
		"bogus-operation")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
	if requestMade {
		t.Error("a request was made despite the operation being invalid")
	}
}
