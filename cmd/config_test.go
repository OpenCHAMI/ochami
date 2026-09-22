// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// config_test.go exercises the "config" commands, which read and write real
// config files. Each test uses a temporary config file supplied via --config so
// the user's real configuration is never touched. These commands do not make
// network requests. Rejection-path cases are covered in config_errors_test.go.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTempConfig creates an (empty) YAML config file in a temp dir and returns
// its path. Pre-creating the file avoids the interactive "create it?" prompt in
// commands that write config.
func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	return path
}

// TestConfigSet_ThenShow verifies that "config set" persists a key to the given
// config file and "config show" reads it back.
func TestConfigSet_ThenShow(t *testing.T) {
	cfg := writeTempConfig(t, "")

	// Set a value.
	setRes := runOchami(t, "--config", cfg, "config", "set", "log.format", "json")
	if setRes.err != nil {
		t.Fatalf("config set: unexpected error: %v (exit %d)", setRes.err, setRes.exitCode)
	}

	// The file should now contain the value.
	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("failed to read config back: %v", err)
	}
	if !strings.Contains(string(data), "json") {
		t.Errorf("config file = %q, want it to contain the set value", string(data))
	}

	// Show the specific key back.
	showRes := runOchami(t, "--config", cfg, "config", "show", "log.format")
	if showRes.err != nil {
		t.Fatalf("config show: unexpected error: %v (exit %d)", showRes.err, showRes.exitCode)
	}
	if !strings.Contains(showRes.stdout, "json") {
		t.Errorf("config show stdout = %q, want it to contain json", showRes.stdout)
	}
}

// TestConfigUnset_Success verifies that "config unset" removes a previously-set key.
func TestConfigUnset_Success(t *testing.T) {
	cfg := writeTempConfig(t, "log:\n  format: json\n")

	res := runOchami(t, "--config", cfg, "config", "unset", "log.format")
	if res.err != nil {
		t.Fatalf("config unset: unexpected error: %v (exit %d)", res.err, res.exitCode)
	}

	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("failed to read config back: %v", err)
	}
	// After unsetting, the format value should be gone.
	if strings.Contains(string(data), "json") {
		t.Errorf("config file = %q, want the unset value to be gone", string(data))
	}
}

// TestConfigClusterSet_ThenShow verifies that "config cluster set" adds a cluster
// entry and "config cluster show" reads it back.
func TestConfigClusterSet_ThenShow(t *testing.T) {
	cfg := writeTempConfig(t, "")

	setRes := runOchami(t, "--config", cfg, "config", "cluster", "set",
		"foobar", "cluster.uri", "https://foobar.openchami.cluster")
	if setRes.err != nil {
		t.Fatalf("config cluster set: unexpected error: %v (exit %d)", setRes.err, setRes.exitCode)
	}

	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("failed to read config back: %v", err)
	}
	if !strings.Contains(string(data), "foobar") {
		t.Errorf("config file = %q, want it to contain the cluster name", string(data))
	}

	showRes := runOchami(t, "--config", cfg, "config", "cluster", "show", "foobar")
	if showRes.err != nil {
		t.Fatalf("config cluster show: unexpected error: %v (exit %d)", showRes.err, showRes.exitCode)
	}
	if !strings.Contains(showRes.stdout, "foobar.openchami.cluster") {
		t.Errorf("config cluster show stdout = %q, want it to contain the URI", showRes.stdout)
	}
}

// TestConfigShow_DefaultedKey verifies that "config show <key>" returns the
// koanf-applied default for a key that is absent from the file. Here the file
// sets only log.level, so log.format should come back as its default rather
// than empty.
func TestConfigShow_DefaultedKey(t *testing.T) {
	cfg := writeTempConfig(t, "log:\n  level: debug\n")

	res := runOchami(t, "--config", cfg, "config", "show", "log.format")
	if res.err != nil {
		t.Fatalf("config show: unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	// The default log format is non-empty; assert we got a value back rather
	// than an empty string (the exact default is owned by the config package).
	if strings.TrimSpace(res.stdout) == "" {
		t.Errorf("config show log.format stdout = %q, want a defaulted (non-empty) value", res.stdout)
	}
}

// TestConfigShow_WholeConfig verifies that "config show" with no key prints the
// merged configuration, including defaulted values.
func TestConfigShow_WholeConfig(t *testing.T) {
	cfg := writeTempConfig(t, "log:\n  level: debug\n")

	res := runOchami(t, "--config", cfg, "config", "show")
	if res.err != nil {
		t.Fatalf("config show: unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	// The explicitly-set value and a defaulted section should both appear.
	if !strings.Contains(res.stdout, "debug") {
		t.Errorf("config show stdout = %q, want it to contain the explicitly-set log level", res.stdout)
	}
	if !strings.Contains(res.stdout, "log:") {
		t.Errorf("config show stdout = %q, want it to contain the log section", res.stdout)
	}
}

// TestConfigClusterUnset_Success verifies that "config cluster unset" removes a key from
// an existing cluster entry in the config file.
func TestConfigClusterUnset_Success(t *testing.T) {
	// Seed a config file with a cluster that has a uri and a smd uri.
	cfg := writeTempConfig(t, `clusters:
  - name: foobar
    cluster:
      uri: https://foobar.openchami.cluster
      smd:
        uri: /hsm/v2
`)

	res := runOchami(t, "--config", cfg, "config", "cluster", "unset", "foobar", "cluster.smd.uri")
	if res.err != nil {
		t.Fatalf("config cluster unset: unexpected error: %v (exit %d)", res.err, res.exitCode)
	}

	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("failed to read config back: %v", err)
	}
	// The removed key's value should be gone; the cluster itself should remain.
	if strings.Contains(string(data), "/hsm/v2") {
		t.Errorf("config file = %q, want the unset key to be gone", string(data))
	}
	if !strings.Contains(string(data), "foobar") {
		t.Errorf("config file = %q, want the cluster to remain", string(data))
	}
}

// TestConfigClusterDelete_Success verifies that "config cluster delete" removes a whole
// cluster entry from the config file.
func TestConfigClusterDelete_Success(t *testing.T) {
	cfg := writeTempConfig(t, `clusters:
  - name: foobar
    cluster:
      uri: https://foobar.openchami.cluster
`)

	res := runOchami(t, "--config", cfg, "config", "cluster", "delete", "foobar")
	if res.err != nil {
		t.Fatalf("config cluster delete: unexpected error: %v (exit %d)", res.err, res.exitCode)
	}

	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatalf("failed to read config back: %v", err)
	}
	if strings.Contains(string(data), "foobar") {
		t.Errorf("config file = %q, want the deleted cluster to be gone", string(data))
	}
}
