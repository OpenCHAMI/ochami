// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_test.go exercises "smd service status" end-to-end against an
// httptest.Server, asserting the outbound request method/path and the
// resolved exit code. Each SMD resource's commands are covered in its own
// smd_<resource>_test.go file, and HTTP-failure cases in smd_errors_test.go.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSMDServiceStatus_Success verifies "smd service status" issues a GET under /service.
func TestSMDServiceStatus_Success(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Write([]byte(`{"code":0}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "smd", "service", "status", "--ignore-config", "--uri", srv.URL)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.HasPrefix(gotPath, "/service") {
		t.Errorf("path = %q, want a /service path", gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
}
