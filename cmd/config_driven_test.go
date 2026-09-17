// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// config_driven_test.go drives config-file-dependent code paths end-to-end: the
// default-cluster and per-service URI resolution in GetBaseURI, and
// HandleToken's enable-auth branch (which reads and validates a token from a
// cluster-scoped environment variable).

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestDefaultClusterURIResolution verifies a command resolves its base URI from
// the default cluster's cluster.uri in a config file (no --uri flag).
func TestDefaultClusterURIResolution(t *testing.T) {
	var hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	cfg := writeTempConfig(t, `default-cluster: demo
clusters:
- name: demo
  cluster:
    uri: `+srv.URL+`
    enable-auth: false
`)

	res := runOchami(t, "--config", cfg, "smd", "group", "get")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !hit {
		t.Error("expected the server (from default-cluster uri) to be contacted")
	}
}

// TestPerServiceURIOverride verifies a per-service URI override in the cluster
// config is honored.
func TestPerServiceURIOverride(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	cfg := writeTempConfig(t, `default-cluster: demo
clusters:
- name: demo
  cluster:
    uri: https://unused.example.com
    smd:
      uri: `+srv.URL+`/smd
    enable-auth: false
`)

	res := runOchami(t, "--config", cfg, "smd", "group", "get")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.HasPrefix(gotPath, "/smd") {
		t.Errorf("path = %q, want the /smd override", gotPath)
	}
}

// TestEnableAuthReadsTokenFromEnv verifies HandleToken's enable-auth branch:
// with enable-auth true and a valid <CLUSTER>_ACCESS_TOKEN env var, the token
// is read and validated and the request succeeds.
func TestEnableAuthReadsTokenFromEnv(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	cfg := writeTempConfig(t, `default-cluster: demo
clusters:
- name: demo
  cluster:
    uri: `+srv.URL+`
    enable-auth: true
`)

	tok := validToken(t)
	t.Setenv("DEMO_ACCESS_TOKEN", tok)

	res := runOchami(t, "--config", cfg, "smd", "group", "get")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(gotAuth, "Bearer") {
		t.Errorf("Authorization header = %q, want it to carry the bearer token", gotAuth)
	}
}

// TestEnableAuthMissingTokenFails verifies that with enable-auth true and no
// token available, the command fails with CodeAuth.
func TestEnableAuthMissingTokenFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	cfg := writeTempConfig(t, `default-cluster: demo
clusters:
- name: demo
  cluster:
    uri: `+srv.URL+`
    enable-auth: true
`)

	// Ensure the env var is not set.
	os.Unsetenv("DEMO_ACCESS_TOKEN")

	res := runOchami(t, "--config", cfg, "smd", "group", "get")
	if res.err == nil {
		t.Fatal("expected an auth error, got nil")
	}
	if res.exitCode != cli.CodeAuth {
		t.Errorf("exit code = %d, want %d (CodeAuth)", res.exitCode, cli.CodeAuth)
	}
}

// TestEnableAuthDisabledSkipsToken verifies that with enable-auth false, no
// token is required or sent.
func TestEnableAuthDisabledSkipsToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	cfg := writeTempConfig(t, `default-cluster: demo
clusters:
- name: demo
  cluster:
    uri: `+srv.URL+`
    enable-auth: false
`)

	res := runOchami(t, "--config", cfg, "smd", "group", "get")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotAuth != "" {
		t.Errorf("Authorization header = %q, want empty (auth disabled)", gotAuth)
	}
}
