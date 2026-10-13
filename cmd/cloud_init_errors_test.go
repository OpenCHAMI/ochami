// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestCloudInitServiceVersion_HTTPError verifies that "cloud-init service
// version" resolves an unsuccessful HTTP response to CodeHTTP.
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
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestCloudInitServiceStatus_HTTPError verifies that "cloud-init service
// status" reports a service that responds with HTTP 500 as running abnormally
// and fails with CodeHTTP.
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
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
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
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
	if res.stdout != "" {
		t.Errorf("stdout = %q, want empty output", res.stdout)
	}
}

// TestCloudInitServiceStatus_APIError verifies that "cloud-init service status
// --api" resolves an unsuccessful HTTP response to CodeHTTP.
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
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeHTTP, cli.CodeName(cli.CodeHTTP))
	}
}

// TestCloudInitServiceStatus_NotRunning verifies that when the service is
// unreachable, "cloud-init service status" reports not running and resolves to
// CodeNetwork.
func TestCloudInitServiceStatus_NotRunning(t *testing.T) {
	t.Parallel()

	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	res := runOchamiWithRuntime(t, "--ignore-config", "cloud-init", "service", "status", "--uri", url)

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
	if !strings.Contains(res.stdout, "cloud-init is not running") {
		t.Errorf("stdout = %q, want it to report not running", res.stdout)
	}
}

// TestCloudInit_MalformedSuccessResponses verifies that the cloud-init group
// and node get commands fail with CodePayload when a successful response can't
// be decoded.
func TestCloudInit_MalformedSuccessResponses(t *testing.T) {
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
				io.WriteString(w, tc.body)
			}))
			defer srv.Close()
			args := append([]string{"--ignore-config"}, tc.args...)
			args = append(args, "--uri", srv.URL, "--token", "t")
			res := runOchamiWithRuntime(t, args...)
			if res.err == nil || res.exitCode != cli.CodePayload {
				t.Fatalf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
			}
		})
	}
}

// TestCloudInitDefaults_GetMalformedResponse verifies that "cloud-init defaults
// get" fails with CodePayload when the response can't be decoded.
func TestCloudInitDefaults_GetMalformedResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"defaults":`))
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "--uri", srv.URL, "--token", "t",
		"cloud-init", "defaults", "get")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
}
