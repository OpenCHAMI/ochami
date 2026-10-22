// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// service_status_test.go exercises per-service "status"/"version" leaf
// commands across the tree.

import (
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

// TestServiceCommands_VersionAndStatusPaths verifies GET path routing for
// several services' version/status leaf commands.
func TestServiceCommands_VersionAndStatusPaths(t *testing.T) {
	t.Parallel()

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
			var gotMethod, gotPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotMethod = r.Method
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, tc.body)
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
			if gotMethod != http.MethodGet {
				t.Errorf("method = %q, want GET", gotMethod)
			}
		})
	}
}

// TestServiceCommands_FalseStatusFlags verifies that status flags passed
// explicitly as false (e.g. --all=false) query the same endpoints as omitting
// them.
func TestServiceCommands_FalseStatusFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      []string
		wantPaths []string
	}{
		{"smd status --all=false", []string{"smd", "service", "status", "--all=false"}, []string{"/service/ready"}},
		{"deprecated smd status --all=false", []string{"smd", "status", "--all=false"}, []string{"/service/ready"}},
		{"bss status --all=false", []string{"bss", "service", "status", "--all=false"}, []string{"/service/status"}},
		{"bss status --storage=false", []string{"bss", "service", "status", "--storage=false"}, []string{"/service/status"}},
		{"bss status --smd=false", []string{"bss", "service", "status", "--smd=false"}, []string{"/service/status"}},
		{"deprecated bss status --all=false", []string{"bss", "status", "--all=false"}, []string{"/service/status"}},
		{"deprecated bss status --storage=false", []string{"bss", "status", "--storage=false"}, []string{"/service/status"}},
		{"deprecated bss status --smd=false", []string{"bss", "status", "--smd=false"}, []string{"/service/status"}},
		{"deprecated bss status --version=false", []string{"bss", "status", "--version=false"}, []string{"/service/status"}},
		{"pcs status --all=false", []string{"pcs", "service", "status", "--token", "t", "--all=false"}, []string{"/readiness"}},
		{"pcs status --storage/--smd/--vault=false", []string{"pcs", "service", "status", "--token", "t",
			"--storage=false", "--smd=false", "--vault=false"}, []string{"/readiness"}},
		{"cloud-init status --api=false", []string{"cloud-init", "service", "status", "--api=false"}, []string{"/version"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotPaths []string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPaths = append(gotPaths, r.URL.Path)
				if r.URL.Path == "/readiness" {
					// PCS reports ready with 204 No Content
					w.WriteHeader(http.StatusNoContent)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, `{}`)
			}))
			defer srv.Close()

			args := append(append([]string{}, tc.args...), "--ignore-config", "--uri", srv.URL)
			res := runOchamiWithRuntime(t, args...)
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
			if !slices.Equal(gotPaths, tc.wantPaths) {
				t.Errorf("paths = %v, want %v", gotPaths, tc.wantPaths)
			}
		})
	}
}
