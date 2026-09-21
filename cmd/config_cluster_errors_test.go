// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestConfigClusterSet_MutuallyExclusiveSources verifies that passing both
// --user and --system is a usage error.
func TestConfigClusterSet_MutuallyExclusiveSources(t *testing.T) {

	res := runOchamiWithRuntime(t, "--ignore-config", "config", "cluster", "set", "--user", "--system",
		"foobar", "cluster.uri", "https://foobar.openchami.cluster")
	if res.err == nil {
		t.Fatal("expected a usage error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestConfigClusterDelete_NotFound verifies that deleting a non-existent cluster
// resolves to a config error.
func TestConfigClusterDelete_NotFound(t *testing.T) {

	cfg := writeTempConfig(t, "clusters: []\n")

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "delete", "does-not-exist")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeConfig {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeConfig, cli.CodeName(cli.CodeConfig))
	}
}

// TestConfigClusterUnset_UnknownKey verifies "config cluster unset" rejects a
// key that does not exist for the named cluster.
func TestConfigClusterUnset_UnknownKey(t *testing.T) {
	cfg := writeTempConfig(t, `clusters:
- name: foobar
  cluster:
    uri: https://foobar.openchami.cluster
`)

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "unset", "foobar", "cluster.smd.uri")
	if res.err == nil || res.exitCode != cli.CodeConfig {
		t.Fatalf("result = (err %v, exit %d), want config error", res.err, res.exitCode)
	}
	if !strings.Contains(res.err.Error(), "does not exist") {
		t.Errorf("error = %q, want missing-key context", res.err)
	}
}

// TestConfigClusterShow_NonexistentCluster verifies showing a cluster that does
// not exist in the config.
func TestConfigClusterShow_NonexistentCluster(t *testing.T) {
	cfg := writeTempConfig(t, "clusters: []\n")

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "show", "does-not-exist")
	if res.exitCode != cli.CodeConfig {
		t.Errorf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, cli.CodeConfig, cli.CodeName(cli.CodeConfig))
	}
}

// TestConfigClusterUnset_Nonexistent verifies unsetting a key on a nonexistent
// cluster fails with CodeConfig.
func TestConfigClusterUnset_Nonexistent(t *testing.T) {
	cfg := writeTempConfig(t, "clusters: []\n")

	res := runOchamiWithRuntime(t, "--config", cfg, "config", "cluster", "unset", "nope", "cluster.uri")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeConfig {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeConfig, cli.CodeName(cli.CodeConfig))
	}
}
