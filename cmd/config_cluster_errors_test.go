// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestConfigClusterDelete_NotFound verifies that deleting a non-existent cluster
// resolves to a config error.
func TestConfigClusterDelete_NotFound(t *testing.T) {
	cfg := writeTempConfig(t, "clusters: []\n")

	res := runOchami(t, "--config", cfg, "config", "cluster", "delete", "does-not-exist")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeConfig {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeConfig, cli.CodeName(cli.CodeConfig))
	}
}
