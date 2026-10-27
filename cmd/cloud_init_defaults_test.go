// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// cloud_init_defaults_test.go exercises the "cloud-init defaults" get and set
// commands end-to-end against an httptest.Server, asserting outbound request
// method/path. Error arms are covered in cloud_init_defaults_errors_test.go.

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCloudInitDefaults_Set verifies "cloud-init defaults set -d <payload>"
// issues POST /admin/cluster-defaults.
func TestCloudInitDefaults_Set(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "defaults", "set", "--uri", srv.URL, "--token", "t",
		"-d", `{"cluster-name":"demo"}`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost || gotPath != "/admin/cluster-defaults" {
		t.Errorf("request = %s %s, want POST /admin/cluster-defaults", gotMethod, gotPath)
	}
}

// TestCloudInitDefaults_Get verifies "cloud-init defaults get" issues GET
// /admin/cluster-defaults.
func TestCloudInitDefaults_Get(t *testing.T) {
	t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Write([]byte(`{"cluster-name":"demo"}`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "defaults", "get", "--uri", srv.URL)

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/admin/cluster-defaults" {
		t.Errorf("path = %q, want /admin/cluster-defaults", gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("method = %q, want GET", gotMethod)
	}
}
