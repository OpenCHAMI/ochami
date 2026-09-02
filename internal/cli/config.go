// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"github.com/knadh/koanf/v2"

	"github.com/openchami/ochami/internal/configfile"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/config"
)

// Runtime configuration state for the CLI. The public pkg/config package is
// deliberately state-independent; the CLI keeps its single "effective"
// configuration here and publishes it only after a successful load. Keeping
// this state in the CLI (rather than in pkg/config) lets external consumers use
// pkg/config without inheriting process-global configuration.
var (
	// activeConfig is the effective configuration in use by the current
	// command invocation.
	activeConfig config.Config

	// activeKoanf is the effective merged koanf that produced activeConfig.
	// It backs the read-only "config show" commands when no source flag is
	// given. It may be nil until a load has occurred.
	activeKoanf *koanf.Koanf

	// UserConfigFile is the resolved path to the per-user config file. It is
	// used by the "config" editing commands to target the user config.
	UserConfigFile string
)

// ActiveConfig returns the effective configuration currently in use.
func ActiveConfig() config.Config { return activeConfig }

// ActiveKoanf returns the effective merged koanf backing the active config, or
// nil if no configuration has been loaded.
func ActiveKoanf() *koanf.Koanf { return activeKoanf }

// SetActiveConfig replaces the effective configuration. It is exported for
// tests that need to seed configuration without loading a file.
func SetActiveConfig(c config.Config) { activeConfig = c }

// earlyLogger adapts log.EarlyLogger to the config.Logger interface so that
// verbose configuration tracing continues to honor the --verbose flag.
type earlyLogger struct{}

func (earlyLogger) Logf(format string, args ...any) {
	log.EarlyLogger.BasicLogf(format, args...)
}

// loadDefaultConfig loads only the built-in defaults (used for
// --ignore-config).
func loadDefaultConfig() error {
	cfg, err := config.LoadDefaults(config.WithLogger(earlyLogger{}))
	if err != nil {
		return err
	}
	ko, err := configfile.EffectiveKoanf()
	if err != nil {
		return err
	}
	activeConfig = cfg
	activeKoanf = ko
	return nil
}

// loadMergedConfig loads the built-in defaults merged with the optional system
// and user config files, publishing the result as the active configuration.
func loadMergedConfig() error {
	userPath, err := config.UserConfigPath()
	if err != nil {
		return err
	}
	UserConfigFile = userPath

	cfg, err := config.Load([]config.Source{
		{Name: "system", Path: config.SystemConfigFile, Optional: true},
		{Name: "user", Path: userPath, Optional: true},
	}, config.WithLogger(earlyLogger{}))
	if err != nil {
		return err
	}
	ko, err := configfile.EffectiveKoanf(config.SystemConfigFile, userPath)
	if err != nil {
		return err
	}
	activeConfig = cfg
	activeKoanf = ko
	return nil
}

// loadConfigFromFile loads the built-in defaults merged with a single required
// config file, publishing the result as the active configuration.
func loadConfigFromFile(path string) error {
	cfg, err := config.LoadFile(path, config.WithLogger(earlyLogger{}))
	if err != nil {
		return err
	}
	ko, err := configfile.ReadConfigWithDefaults(path)
	if err != nil {
		return err
	}
	activeConfig = cfg
	activeKoanf = ko
	return nil
}
