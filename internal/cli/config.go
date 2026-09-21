// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"github.com/knadh/koanf/v2"

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

// configLoadOpts returns the LoadOptions the CLI's config loaders should use.
// A logger is attached only when --verbose is actually on, so that
// pkg/config's trace path (which enumerates and formats every config key) is
// skipped entirely on the common, non-verbose invocation instead of being
// built and then discarded.
func configLoadOpts() []config.LoadOption {
	if !log.EarlyLogger.EarlyVerbose {
		return nil
	}
	return []config.LoadOption{config.WithLogger(earlyLogger{})}
}

// loadDefaultConfig loads only the built-in defaults (used for
// --ignore-config).
func loadDefaultConfig() error {
	ko, cfg, err := config.LoadDefaultsKoanf(configLoadOpts()...)
	if err != nil {
		return err
	}
	activeConfig = cfg
	activeKoanf = ko
	// Resolve (but do not read) the user config path even under
	// --ignore-config, so commands that report or target it (e.g. "config
	// show --user") still have a usable path instead of an empty string.
	if UserConfigFile == "" {
		userPath, err := config.UserConfigPath()
		if err != nil {
			return err
		}
		UserConfigFile = userPath
	}
	return nil
}

// loadMergedConfig loads the built-in defaults merged with the optional system
// and user config files, publishing the result as the active configuration.
// activeConfig and activeKoanf are derived from a single underlying load (via
// config.LoadMergedKoanf) so the two can never observe different file
// contents, and --verbose tracing covers the load that actually produced them.
func loadMergedConfig() error {
	userPath, err := config.UserConfigPath()
	if err != nil {
		return err
	}
	UserConfigFile = userPath

	ko, cfg, err := config.LoadMergedKoanf(configLoadOpts()...)
	if err != nil {
		return err
	}
	activeConfig = cfg
	activeKoanf = ko
	return nil
}

// loadConfigFromFile loads the built-in defaults merged with a single required
// config file, publishing the result as the active configuration. activeConfig
// and activeKoanf are derived from a single underlying load (via
// config.LoadFileKoanf) so the two can never observe different file contents.
func loadConfigFromFile(path string) error {
	ko, cfg, err := config.LoadFileKoanf(path, configLoadOpts()...)
	if err != nil {
		return err
	}
	activeConfig = cfg
	activeKoanf = ko
	return nil
}
