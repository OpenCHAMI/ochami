// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// config_errors_test.go covers rejection-path cases for the "config"
// commands; see config_test.go (including the writeTempConfig helper) for
// the success paths these mirror.

import (
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestConfigSet_RejectsClusterKey verifies that "config set" refuses to modify
// cluster config (which belongs to "config cluster set") and reports a usage
// error.
func TestConfigSet_RejectsClusterKey(t *testing.T) {
	cfg := writeTempConfig(t, "")

	res := runOchami(t, "--config", cfg, "config", "set", "clusters.foo", "bar")

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}

// TestConfigShow_RejectsClusterKey verifies that "config show clusters.<name>" is
// rejected with a config error, since cluster keys must be read via
// "config cluster show".
func TestConfigShow_RejectsClusterKey(t *testing.T) {
	cfg := writeTempConfig(t, "")

	res := runOchami(t, "--config", cfg, "config", "show", "clusters.foo")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeConfig {
		t.Errorf("exit code = %d, want %d (CodeConfig)", res.exitCode, cli.CodeConfig)
	}
}

// TestConfigClusterDelete_NotFound verifies that deleting a non-existent cluster
// resolves to a config error.
func TestConfigClusterDelete_NotFound(t *testing.T) {
	cfg := writeTempConfig(t, "clusters: []\n")

	res := runOchami(t, "--config", cfg, "config", "cluster", "delete", "does-not-exist")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeConfig {
		t.Errorf("exit code = %d, want %d (CodeConfig)", res.exitCode, cli.CodeConfig)
	}
}
