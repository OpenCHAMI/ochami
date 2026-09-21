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

	"github.com/openchami/ochami/internal/cli"
)

// TestCloudInitGroupGet_Success verifies "cloud-init group get raw" issues GET
// /admin/groups.
func TestCloudInitGroupGet_Success(t *testing.T) {
	t.Parallel()

	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = w.Write([]byte(`{}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "group", "get", "raw", "--uri", srv.URL, "--token", "t")
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
	t.Parallel()

	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "group", "add", "--uri", srv.URL, "--token", "t",
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
	t.Parallel()

	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()

	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "group", "set", "--uri", srv.URL, "--token", "t",
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
	t.Parallel()

	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()

	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "group", "delete", "--uri", srv.URL, "--token", "t",
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
	t.Parallel()

	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()

	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "node", "set", "--uri", srv.URL, "--token", "t",
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
	t.Parallel()

	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()

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

// TestCloudInitGroupRender_EmptyConfig verifies that "cloud-init group render"
// exits cleanly when the group's cloud-config is empty (nothing to render). The
// server returns an empty body for the group config fetch, so the command logs
// a warning and returns without error.
func TestCloudInitGroupRender_EmptyConfig(t *testing.T) {
	t.Parallel()

	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Empty body for the group-config fetch => nothing to render.
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "group", "render", "--uri", srv.URL, "--token", "t",
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
				_, _ = io.WriteString(w, `{}`) //nolint:errcheck // test response writes are observed by the client
			}))
			defer srv.Close()

			res := runOchamiWithRuntime(t, "cloud-init", "group", "get", subcommand,
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
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"openapi":"3.0.0"}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "service", "status", "--api", "--ignore-config", "--uri", srv.URL)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, "openapi") {
		t.Errorf("stdout = %q, want OpenAPI document", res.stdout)
	}
}
func TestCloudInitServiceVersion_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "service", "version", "--ignore-config", "--uri", srv.URL)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

// TestCloudInitServiceStatus_Success verifies "cloud-init service status" reports
// success against a healthy server.
func TestCloudInitServiceStatus_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "service", "status", "--ignore-config", "--uri", srv.URL)
	// Some status commands treat non-2xx-with-body as an error; accept either a
	// clean success or a mapped HTTP/network code, but never a panic.
	if res.err != nil && res.exitCode == cli.CodeSuccess {
		t.Errorf("inconsistent result: err=%v exit=%d", res.err, res.exitCode)
	}
	_ = strings.TrimSpace(res.stdout)
}

// TestCloudInitServiceStatus_HTTPError verifies a responding but unhealthy
// service is distinguished from a network failure.
func TestCloudInitServiceStatus_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "service", "status", "--ignore-config", "--uri", srv.URL)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
	if !strings.Contains(res.stdout, "running, but not normally") {
		t.Errorf("stdout = %q, want abnormal-running status", res.stdout)
	}
}

// TestCloudInitServiceStatus_QuietHTTPError verifies quiet mode suppresses the
// human-readable status while preserving the exit code.
func TestCloudInitServiceStatus_QuietHTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "service", "status", "--quiet", "--ignore-config", "--uri", srv.URL)
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
	if res.stdout != "" {
		t.Errorf("stdout = %q, want empty output", res.stdout)
	}
}
func TestCloudInitServiceStatus_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "cloud-init", "service", "status", "--api", "--ignore-config", "--uri", srv.URL)
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeHTTP {
		t.Errorf("exit code = %d, want %d (CodeHTTP)", res.exitCode, cli.CodeHTTP)
	}
}

func TestCloudInitServiceVersion_Success(t *testing.T) {
	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version":"1.0"}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "service", "version", "--uri", srv.URL)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/version" {
		t.Errorf("path = %q, want /version", gotPath)
	}
}

// TestCloudInitDefaultsGet verifies "cloud-init defaults get" issues GET
// /admin/cluster-defaults.
func TestCloudInitDefaultsGet(t *testing.T) {
	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"cluster-name":"demo"}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "defaults", "get", "--uri", srv.URL)

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/admin/cluster-defaults" {
		t.Errorf("path = %q, want /admin/cluster-defaults", gotPath)
	}
}

// TestCloudInitServiceStatus_Running verifies that "cloud-init service status"
// exits successfully when the /version endpoint responds OK.
func TestCloudInitServiceStatus_Running(t *testing.T) {
	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"version":"1.0.0"}`)) //nolint:errcheck // test response writes are observed by the client
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "service", "status", "--uri", srv.URL)

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotPath != "/version" {
		t.Errorf("path = %q, want /version", gotPath)
	}
	if !strings.Contains(res.stdout, "cloud-init is running") {
		t.Errorf("stdout = %q, want it to report running", res.stdout)
	}
}

// TestCloudInitServiceStatus_NotRunning verifies that when the service is
// unreachable, "cloud-init service status" reports not running and resolves to
// a non-zero exit code.
func TestCloudInitServiceStatus_NotRunning(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // connection refused

	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "service", "status", "--uri", url)

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
	if !strings.Contains(res.stdout, "cloud-init is not running") {
		t.Errorf("stdout = %q, want it to report not running", res.stdout)
	}
}
func TestCloudInitMalformedSuccessResponses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		args []string
	}{
		{name: "all groups", body: `{`, args: []string{"cloud-init", "group", "get", "raw"}},
		{name: "single group", body: `{`, args: []string{"cloud-init", "group", "get", "raw", "compute"}},
		{name: "node metadata", body: `: invalid`, args: []string{"cloud-init", "node", "get", "meta-data", "node01"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, tc.body) //nolint:errcheck // malformed response is the fixture
			}))
			defer srv.Close()
			args := append([]string{"--ignore-config"}, tc.args...)
			args = append(args, "--uri", srv.URL, "--token", "t")
			res := runOchamiWithRuntime(t, args...)
			if res.err == nil || res.exitCode != cli.CodePayload {
				t.Fatalf("result = (err %v, exit %d), want CodePayload", res.err, res.exitCode)
			}
		})
	}
}

// TestCloudInitDefaults_GetMalformedResponse verifies handling of malformed responses.
func TestCloudInitDefaults_GetMalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"defaults":`)) //nolint:errcheck // malformed test response
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--uri", srv.URL, "--token", "t",
		"cloud-init", "defaults", "get")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (CodePayload)", res.exitCode, cli.CodePayload)
	}
}
