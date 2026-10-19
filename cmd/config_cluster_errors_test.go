// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/config"
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

// duplicateClusterConfig defines cluster "demo" twice, as a manual edit
// might.
const duplicateClusterConfig = `default-cluster: demo
clusters:
  - name: demo
    cluster:
      uri: https://first.example
      bss:
        uri: /first
  - name: demo
    cluster:
      uri: https://second.example
      bss:
        uri: /second
`

// requireDuplicateClusterRejected runs ochami with args against a config file
// that defines cluster "demo" twice and fails t unless the command fails with
// ErrDuplicateCluster and a config error exit code and leaves the file
// unchanged.
func requireDuplicateClusterRejected(t *testing.T, args ...string) {
	t.Helper()
	cfg := writeTempConfig(t, duplicateClusterConfig)

	res := runOchami(t, append([]string{"--config", cfg}, args...)...)
	if !errors.As(res.err, &config.ErrDuplicateCluster{}) {
		t.Fatalf("error = %v, want ErrDuplicateCluster", res.err)
	}
	if res.exitCode != cli.CodeConfig {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeConfig, cli.CodeName(cli.CodeConfig))
	}
	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("failed to read config back: %v", err)
	}
	if !bytes.Equal(data, []byte(duplicateClusterConfig)) {
		t.Errorf("config file changed to:\n%s", data)
	}
}

// TestConfigClusterSet_DuplicateCluster verifies that setting a key on,
// renaming, or making default a cluster defined more than once in the config
// file fails with a config error and leaves the file unchanged.
func TestConfigClusterSet_DuplicateCluster(t *testing.T) {
	t.Run("key", func(t *testing.T) {
		requireDuplicateClusterRejected(t, "config", "cluster", "set", "demo", "cluster.uri", "https://new.example")
	})
	t.Run("rename", func(t *testing.T) {
		requireDuplicateClusterRejected(t, "config", "cluster", "set", "demo", "name", "prod")
	})
	t.Run("default", func(t *testing.T) {
		requireDuplicateClusterRejected(t, "config", "cluster", "set", "-d", "demo", "cluster.uri", "https://new.example")
	})
}

// TestConfigClusterUnset_DuplicateCluster verifies that unsetting a key from a
// cluster defined more than once in the config file fails with a config error
// and leaves the file unchanged.
func TestConfigClusterUnset_DuplicateCluster(t *testing.T) {
	requireDuplicateClusterRejected(t, "config", "cluster", "unset", "demo", "cluster.bss.uri")
}

// TestConfigClusterDelete_DuplicateCluster verifies that deleting a cluster
// defined more than once in the config file fails with a config error and
// leaves the file unchanged.
func TestConfigClusterDelete_DuplicateCluster(t *testing.T) {
	requireDuplicateClusterRejected(t, "config", "cluster", "delete", "demo")
}
