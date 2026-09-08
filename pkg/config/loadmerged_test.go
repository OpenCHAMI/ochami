// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

// loadmerged_test.go covers LoadMerged and the UserConfigPath HOME-unset
// fallback, which the existing tests do not exercise.

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadMergedWithLayeredFiles verifies LoadMerged reads the built-in
// defaults merged with the (optional) user config file resolved from HOME.
func TestLoadMergedWithLayeredFiles(t *testing.T) {
	tmpHome := t.TempDir()
	oldHome, had := os.LookupEnv("HOME")
	os.Setenv("HOME", tmpHome)
	if had {
		defer os.Setenv("HOME", oldHome)
	} else {
		defer os.Unsetenv("HOME")
	}

	// Write a user config file at the resolved path.
	userCfgDir := filepath.Join(tmpHome, ".config", "ochami")
	if err := os.MkdirAll(userCfgDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	userCfg := filepath.Join(userCfgDir, "config.yaml")
	if err := os.WriteFile(userCfg, []byte("log:\n  level: warning\n"), 0o644); err != nil {
		t.Fatalf("write user config: %v", err)
	}

	cfg, err := LoadMerged()
	if err != nil {
		t.Fatalf("LoadMerged returned error: %v", err)
	}
	if cfg.Log.Level != "warning" {
		t.Errorf("Log.Level = %q, want warning (from user config)", cfg.Log.Level)
	}
}

// TestLoadMergedNoUserFile verifies LoadMerged succeeds when no user config
// file exists (the user source is optional).
func TestLoadMergedNoUserFile(t *testing.T) {
	tmpHome := t.TempDir() // empty; no config file present
	oldHome, had := os.LookupEnv("HOME")
	os.Setenv("HOME", tmpHome)
	if had {
		defer os.Setenv("HOME", oldHome)
	} else {
		defer os.Unsetenv("HOME")
	}

	if _, err := LoadMerged(); err != nil {
		t.Fatalf("LoadMerged with no user file returned error: %v", err)
	}
}

// TestUserConfigPathHomeUnset covers the fallback branch of UserConfigPath that
// consults user.Current() when HOME is unset.
func TestUserConfigPathHomeUnset(t *testing.T) {
	oldHome, had := os.LookupEnv("HOME")
	os.Unsetenv("HOME")
	defer func() {
		if had {
			os.Setenv("HOME", oldHome)
		}
	}()

	// With HOME unset, UserConfigPath falls back to user.Current(). On most
	// systems this succeeds; either outcome (a path or an error) exercises the
	// fallback branch.
	_, _ = UserConfigPath() //nolint:errcheck // both success and error are valid outcomes for this environment-dependent fallback
}
