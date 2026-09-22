// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
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
	// The file sets only log.level, so log.format must come from the defaults.
	if !strings.Contains(res.stdout, "debug") {
		t.Errorf("config show stdout = %q, want it to contain the explicitly-set log level", res.stdout)
	}
	if !strings.Contains(res.stdout, "format:") {
		t.Errorf("config show stdout = %q, want it to contain the defaulted log format", res.stdout)
	}
}
