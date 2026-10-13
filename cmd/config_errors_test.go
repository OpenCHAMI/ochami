// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// config_errors_test.go covers rejection-path cases for the "config"
// commands; see config_test.go (including the writeTempConfig helper) for
// the success paths these mirror.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestConfigSet_RejectsClusterKey verifies that "config set" refuses to modify
// cluster config (which belongs to "config cluster set") and reports a usage
// error.
func TestConfigSet_RejectsClusterKey(t *testing.T) {
	t.Parallel()

	cfg := writeTempConfig(t, "")

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "set", "clusters.foo", "bar")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestConfigShow_RejectsClusterKey verifies that "config show clusters.<name>" is
// rejected with a config error, since cluster keys must be read via
// "config cluster show".
func TestConfigShow_RejectsClusterKey(t *testing.T) {
	t.Parallel()

	cfg := writeTempConfig(t, "")

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "show", "clusters.foo")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeConfig {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeConfig, cli.CodeName(cli.CodeConfig))
	}
}

// TestEnableAuth_MissingTokenFails verifies that with enable-auth true and no
// token available, the command fails with CodeAuth.
func TestEnableAuth_MissingTokenFails(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	cfg := writeTempConfig(t, `default-cluster: demo
clusters:
- name: demo
  cluster:
    uri: `+srv.URL+`
    enable-auth: true
`)

	// runOchamiWithRuntime's environment is empty, so DEMO_ACCESS_TOKEN is
	// unset regardless of the host's environment.
	res := runOchamiWithRuntime(t, "--config", cfg, "smd", "group", "get")
	if res.err == nil {
		t.Fatal("expected an auth error, got nil")
	}
	if res.exitCode != cli.CodeAuth {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeAuth, cli.CodeName(cli.CodeAuth))
	}
}

// TestConfigUnset_UnknownKey verifies "config unset" rejects a key that does
// not exist in the config file.
func TestConfigUnset_UnknownKey(t *testing.T) {
	t.Parallel()

	cfg := writeTempConfig(t, "log:\n  format: json\n")

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "unset", "log.does-not-exist")
	if res.err == nil || res.exitCode != cli.CodeConfig {
		t.Fatalf("result = (err %v, exit %d), want config error", res.err, res.exitCode)
	}
	if !strings.Contains(res.err.Error(), "does not exist") {
		t.Errorf("error = %q, want missing-key context", res.err)
	}
}
