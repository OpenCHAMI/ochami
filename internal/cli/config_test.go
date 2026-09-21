// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"os"
	"testing"

	"github.com/openchami/ochami/internal/log"
)

// TestLoadDefaultConfig_ResolvesUserConfigFile verifies that loadDefaultConfig
// (used for --ignore-config) still resolves UserConfigFile, without reading
// it, so commands that report or target it (e.g. "config show --user") still
// have a usable path instead of an empty string.
func TestLoadDefaultConfig_ResolvesUserConfigFile(t *testing.T) {
	origUserConfigFile := UserConfigFile
	origConfig := ActiveConfig()
	origKoanf := activeKoanf
	t.Cleanup(func() {
		UserConfigFile = origUserConfigFile
		SetActiveConfig(origConfig)
		activeKoanf = origKoanf
	})
	UserConfigFile = ""

	tmpHome := t.TempDir()
	oldHome, had := os.LookupEnv("HOME")
	os.Setenv("HOME", tmpHome)
	t.Cleanup(func() {
		if had {
			os.Setenv("HOME", oldHome)
		} else {
			os.Unsetenv("HOME")
		}
	})

	if err := loadDefaultConfig(); err != nil {
		t.Fatalf("loadDefaultConfig() error = %v", err)
	}
	if UserConfigFile == "" {
		t.Error("loadDefaultConfig() did not resolve UserConfigFile")
	}
}

// TestConfigLoadOpts verifies that configLoadOpts attaches a logger only when
// --verbose (log.EarlyLogger.EarlyVerbose) is on, so pkg/config's
// key-enumeration trace path is only paid for when it will actually be
// observed.
func TestConfigLoadOpts(t *testing.T) {
	orig := log.EarlyLogger.EarlyVerbose
	t.Cleanup(func() { log.EarlyLogger.EarlyVerbose = orig })

	log.EarlyLogger.EarlyVerbose = false
	if opts := configLoadOpts(); opts != nil {
		t.Errorf("configLoadOpts() = %v, want nil when not verbose", opts)
	}

	log.EarlyLogger.EarlyVerbose = true
	if opts := configLoadOpts(); len(opts) == 0 {
		t.Error("configLoadOpts() returned no options when verbose")
	}
}
