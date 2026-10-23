// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestRootCommand_RejectsCACertWithInsecure verifies that passing --cacert
// and --insecure together is a usage error. The two are contradictory:
// --insecure skips certificate verification entirely, while UseCACert always
// re-enables it once a CA is loaded, so allowing both would silently make
// --insecure a no-op whenever --cacert is also set.
func TestRootCommand_RejectsCACertWithInsecure(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "--ignore-config", "--insecure", "--cacert", "/path/to/ca.pem", "version")
	if res.err == nil {
		t.Fatal("passing --cacert and --insecure together = nil error, want usage error")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
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
