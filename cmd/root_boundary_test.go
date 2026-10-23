// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// root_boundary_test.go exercises edge cases of how the root command loads the
// config file: a --config path skipped by --ignore-config, and an explicit
// null value.

import (
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestRootCommand_NullValueHandling verifies that an explicit YAML "null" for a
// non-nullable key (log.level) is rejected as a config validation error at
// load time, rather than silently resolving to its default.
func TestRootCommand_NullValueHandling(t *testing.T) {
	t.Parallel()

	cfg := writeTempConfig(t, `log:
  level: null
  format: json
clusters:
  - name: test
    cluster:
      uri: http://localhost:8080
`)

	// --ignore-config is deliberately omitted: it would skip reading --config
	// entirely (see InitConfig), which would make this test pass regardless
	// of what the file contains.
	res := runOchamiWithRuntime(t, "--config", cfg, "version")

	if res.err == nil {
		t.Fatal("expected an error for a null log.level, got nil")
	}
	if res.exitCode != cli.CodeConfig {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeConfig, cli.CodeName(cli.CodeConfig))
	}
}

// TestRootCommand_IgnoreConfigSkipsConfigFile verifies that --ignore-config skips
// reading the path given to --config entirely (per InitConfig), so a
// nonexistent path doesn't cause an error.
func TestRootCommand_IgnoreConfigSkipsConfigFile(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "--config", "/nonexistent/path/to/config.yaml", "--ignore-config", "version")

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if res.exitCode != cli.CodeSuccess {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeSuccess, cli.CodeName(cli.CodeSuccess))
	}
}
