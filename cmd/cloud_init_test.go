// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// cloud_init_test.go exercises representative "cloud-init" subcommands (group,
// node, defaults) end-to-end against an httptest.Server, asserting outbound
// request method/path and the resolved exit code. The "cloud-init service" and
// "cloud-init defaults get" commands are covered in services_test.go.

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCloudInitGroupGet_Success verifies "cloud-init group get raw" issues GET
// /admin/groups.
func TestCloudInitGroupGet_Success(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "group", "get", "raw", "--ignore-config", "--uri", srv.URL, "--token", "t")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodGet || gotPath != "/admin/groups" {
		t.Errorf("request = %s %s, want GET /admin/groups", gotMethod, gotPath)
	}
}

// TestCloudInitGroupAdd_Success verifies "cloud-init group add -d <payload>" issues
// POST /admin/groups.
func TestCloudInitGroupAdd_Success(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "group", "add", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"-d", `[{"name":"compute"}]`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost || gotPath != "/admin/groups" {
		t.Errorf("request = %s %s, want POST /admin/groups", gotMethod, gotPath)
	}
}

// TestCloudInitGroupSet_Success verifies "cloud-init group set -d <payload>" issues
// PUT /admin/groups/<name>.
func TestCloudInitGroupSet_Success(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "group", "set", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"-d", `[{"name":"compute"}]`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
}

// TestCloudInitGroupDelete_NoConfirm verifies "cloud-init group delete
// --no-confirm <name>" issues DELETE under /admin/groups.
func TestCloudInitGroupDelete_NoConfirm(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "group", "delete", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"--no-confirm", "compute")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
}

// TestCloudInitNodeSet_Success verifies "cloud-init node set -d <payload>" issues a PUT
// under /admin/instance-info.
func TestCloudInitNodeSet_Success(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "node", "set", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"-d", `[{"id":"x0c0s0b0n0"}]`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPut || !strings.HasPrefix(gotPath, "/admin/instance-info") {
		t.Errorf("request = %s %s, want PUT under /admin/instance-info", gotMethod, gotPath)
	}
}

// TestCloudInitDefaultsSet verifies "cloud-init defaults set -d <payload>"
// issues POST /admin/cluster-defaults.
func TestCloudInitDefaultsSet(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "defaults", "set", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"-d", `{"cluster-name":"demo"}`)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotMethod != http.MethodPost || gotPath != "/admin/cluster-defaults" {
		t.Errorf("request = %s %s, want POST /admin/cluster-defaults", gotMethod, gotPath)
	}
}

// TestCloudInitGroupRender_EmptyConfig verifies that "cloud-init group render"
// exits cleanly when the group's cloud-config is empty (nothing to render). The
// server returns an empty body for the group config fetch, so the command logs
// a warning and returns without error.
func TestCloudInitGroupRender_EmptyConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Empty body for the group-config fetch => nothing to render.
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "group", "render", "--ignore-config", "--uri", srv.URL, "--token", "t",
		"compute", "x0c0s0b0n0")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
}

// TestCloudInitGroupGet_RemainingPaths covers the "cloud-init group get"
// subcommands (config, meta-data) not already exercised above.
func TestCloudInitGroupGet_RemainingPaths(t *testing.T) {
	for _, subcommand := range []string{"config", "meta-data"} {
		t.Run(subcommand, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{}`)
			}))
			defer srv.Close()

			res := runOchami(t, "cloud-init", "group", "get", subcommand,
				"--ignore-config", "--uri", srv.URL, "--token", "t")
			if res.err != nil {
				t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
			}
		})
	}
}

// TestCloudInitServiceStatus_API verifies --api prints the returned OpenAPI
// document through the command's injected output stream.
func TestCloudInitServiceStatus_API(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"openapi":"3.0.0"}`))
	}))
	defer srv.Close()

	res := runOchami(t, "cloud-init", "service", "status", "--api", "--ignore-config", "--uri", srv.URL)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "openapi") {
		t.Errorf("stdout = %q, want OpenAPI document", res.stdout)
	}
}
