// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/config"
)

// TestConfigClusterSet_MutuallyExclusiveSources verifies that passing both
// --user and --system is a usage error.
func TestConfigClusterSet_MutuallyExclusiveSources(t *testing.T) {
	res := runOchami(t, "config", "cluster", "set", "--user", "--system",
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

	res := runOchami(t, "--config", cfg, "config", "cluster", "delete", "does-not-exist")
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

	res := runOchami(t, "--config", cfg, "config", "cluster", "unset", "foobar", "cluster.smd.uri")
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

	res := runOchami(t, "--config", cfg, "config", "cluster", "show", "does-not-exist")
	if res.exitCode != cli.CodeConfig {
		t.Errorf("result = (err %v, exit %d), want %d (%s)", res.err, res.exitCode, cli.CodeConfig, cli.CodeName(cli.CodeConfig))
	}
}

// TestConfigClusterUnset_Nonexistent verifies unsetting a key on a nonexistent
// cluster fails with CodeConfig.
func TestConfigClusterUnset_Nonexistent(t *testing.T) {
	cfg := writeTempConfig(t, "clusters: []\n")

	res := runOchami(t, "--config", cfg, "config", "cluster", "unset", "nope", "cluster.uri")
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

// TestConfigClusterShow_DuplicateCluster verifies that showing a single
// cluster defined more than once in a config file fails with a config error,
// whether reading the merged system and user configuration or a single file.
func TestConfigClusterShow_DuplicateCluster(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// An absolute XDG_CONFIG_HOME would take precedence over HOME.
	t.Setenv("XDG_CONFIG_HOME", "")
	userDir := filepath.Join(home, ".config", "ochami")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatalf("failed to create user config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(userDir, "config.yaml"), []byte(duplicateClusterConfig), 0o644); err != nil {
		t.Fatalf("failed to write user config: %v", err)
	}
	cfg := writeTempConfig(t, duplicateClusterConfig)

	for _, args := range [][]string{
		{"config", "cluster", "show", "demo"},
		{"--config", cfg, "config", "cluster", "show", "demo", "cluster.uri"},
	} {
		res := runOchami(t, args...)
		if !errors.As(res.err, &config.ErrDuplicateCluster{}) {
			t.Errorf("%v: error = %v, want ErrDuplicateCluster", args, res.err)
		}
		if res.exitCode != cli.CodeConfig {
			t.Errorf("%v: exit code = %d, want %d (%s)", args, res.exitCode, cli.CodeConfig, cli.CodeName(cli.CodeConfig))
		}
		if res.stdout != "" {
			t.Errorf("%v: stdout = %q, want nothing shown", args, res.stdout)
		}
	}
}
