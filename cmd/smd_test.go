// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// smd_test.go exercises the commands directly under "smd" (service, status)
// end-to-end against an httptest.Server, asserting outbound request
// method/path and the resolved exit code, and the checks that span several SMD
// resources, such as CA trust for their add commands. Each SMD resource's own
// commands are covered in its smd_<resource>_test.go file, and HTTP-failure
// cases in smd_errors_test.go.

import (
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSMDAdd_TrustsCACert verifies that "smd group add" and "smd rfe add"
// issue their POST to a TLS server whose certificate is trusted only through
// --cacert.
func TestSMDAdd_TrustsCACert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "group", args: []string{"smd", "group", "add", "compute"}},
		{name: "rfe", args: []string{"smd", "rfe", "add", "x0", "n", "172.16.0.1", "de:ad:be:ef:00:00"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var gotMethod string
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				w.WriteHeader(http.StatusCreated)
			}))
			defer srv.Close()

			caPath := filepath.Join(t.TempDir(), "ca.pem")
			caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
			if err := os.WriteFile(caPath, caPEM, 0o600); err != nil {
				t.Fatalf("failed to write CA certificate: %v", err)
			}

			full := append(tc.args, "--ignore-config", "--uri", srv.URL, "--token", "t", "--cacert", caPath)
			res := runOchamiWithRuntime(t, full...)
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if gotMethod != http.MethodPost {
				t.Errorf("method = %q, want POST", gotMethod)
			}
		})
	}
}

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
