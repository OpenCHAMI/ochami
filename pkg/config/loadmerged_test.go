// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

// loadmerged_test.go covers LoadMerged and UserConfigPath's fallback to the
// current user's home directory when HOME is unset.

import (
	"os"
	"os/user"
	"path/filepath"
	"testing"
)

// TestLoadMerged_WithLayeredFiles verifies LoadMerged reads the built-in
// defaults merged with the (optional) user config file resolved from HOME.
func TestLoadMerged_WithLayeredFiles(t *testing.T) {
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

// TestLoadMerged_NoUserFile verifies LoadMerged succeeds when no user config
// file exists (the user source is optional).
func TestLoadMerged_NoUserFile(t *testing.T) {
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

// TestUserConfigPath_HomeUnset verifies UserConfigPath falls back to the
// current user's home directory when HOME is unset.
func TestUserConfigPath_HomeUnset(t *testing.T) {
	oldHome, had := os.LookupEnv("HOME")
	os.Unsetenv("HOME")
	defer func() {
		if had {
			os.Setenv("HOME", oldHome)
		}
	}()

	u, userErr := user.Current()
	got, err := UserConfigPath()
	if userErr != nil {
		// Without a resolvable current user there is nothing to fall back to.
		if err == nil {
			t.Fatalf("UserConfigPath() = %q, want an error when user.Current() fails", got)
		}
		return
	}
	if err != nil {
		t.Fatalf("UserConfigPath() error = %v", err)
	}
	if want := filepath.Join(u.HomeDir, ".config", "ochami", "config.yaml"); got != want {
		t.Errorf("UserConfigPath() = %q, want %q", got, want)
	}
}
