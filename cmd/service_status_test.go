// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// service_status_test.go exercises the remaining per-service "status"/
// "version" leaf commands across the tree that aren't otherwise covered by
// their command family's own test file.

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRemainingServicePaths verifies GET path routing for several services'
// version/status leaf commands.
func TestRemainingServicePaths(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPath string
		body     string
	}{
		{"bss version", []string{"bss", "service", "version"}, "/service/version", `{"version":"1.0"}`},
		{"cloud-init version", []string{"cloud-init", "service", "version"}, "/version", `{"version":"1.0"}`},
		{"metadata status", []string{"metadata", "service", "status"}, "/health", `{}`},
		{"rcs status", []string{"rcs", "service", "status", "--token", "t"}, "/health", `{"status":"ok"}`},
		{"deprecated smd status", []string{"smd", "status"}, "/service/ready", `{}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, tc.body) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			args := append(append([]string{}, tc.args...), "--ignore-config", "--uri", srv.URL)
			res := runOchamiWithRuntime(t, args...)
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if gotPath != tc.wantPath {
				t.Errorf("path = %q, want %q", gotPath, tc.wantPath)
			}
		})
	}
}
