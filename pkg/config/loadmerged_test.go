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

// TestLoadMerged_NoUserFile verifies LoadMerged succeeds when no user config
// file exists (the user source is optional).
func TestLoadMerged_NoUserFile(t *testing.T) {
	tmpHome := t.TempDir() // empty; no config file present
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", tmpHome)

	if _, err := LoadMerged(); err != nil {
		t.Fatalf("LoadMerged with no user file returned error: %v", err)
	}
}

// TestUserConfigPath_HomeUnset verifies that, with HOME unset, UserConfigPath
// falls back to the current user's home directory, or fails if the current user
// can't be determined.
func TestUserConfigPath_HomeUnset(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	// t.Setenv restores HOME when the test ends; unset it until then.
	t.Setenv("HOME", "")
	os.Unsetenv("HOME")

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
