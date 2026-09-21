// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"github.com/knadh/koanf/v2"
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/configfile"
	"github.com/openchami/ochami/pkg/config"
)

// ConfigFileToModify resolves which config file a "config"/"config cluster"
// editing subcommand (set, unset, delete) should target, in precedence order:
//
//  1. rt.ConfigFile, if --config was passed.
//  2. The system config file, if --system was passed.
//  3. Otherwise, the user config file (the default).
//
// cmd.Flag looks up "system" through cmd's inherited persistent flags, so this
// is correct regardless of how deeply cmd is nested under "config" (unlike
// walking a fixed number of cmd.Parent() calls, which silently breaks if a
// command is ever nested at a different depth).
func (rt *Runtime) ConfigFileToModify(cmd *cobra.Command) string {
	if rt.ConfigFile != "" {
		return rt.ConfigFile
	}
	if f := cmd.Flag("system"); f != nil && f.Changed {
		return config.SystemConfigFile
	}
	return rt.UserConfigFile
}

// ResolveShowKoanf resolves the *koanf.Koanf a "config show"/"config cluster
// show" subcommand should read from, in precedence order:
//
//  1. The system config file, if --system was passed.
//  2. The user config file, if --user was passed.
//  3. The file at --config, if passed.
//  4. Otherwise, the runtime's already-loaded effective koanf (rt.Koanf).
//
// Each of the first three cases reads and applies defaults to a single file
// via configfile.ReadConfigWithDefaults; the fourth reflects the merged
// system+user (or --ignore-config default-only) view InitConfig already
// produced.
func (rt *Runtime) ResolveShowKoanf(cmd *cobra.Command) (*koanf.Koanf, error) {
	// cmd.Flag (rather than cmd.Flags().Changed) is used throughout, since it
	// resolves an inherited persistent flag (e.g. "system", defined on the
	// parent "config" command) regardless of nesting depth or of whether
	// Cobra has merged persistent flags into cmd's own FlagSet yet.
	switch {
	case cmd.Flag("system") != nil && cmd.Flag("system").Changed:
		ko, err := configfile.ReadConfigWithDefaults(config.SystemConfigFile)
		if err != nil {
			return nil, Errorf(CodeConfig, "failed to read system config file: %w", err)
		}
		return ko, nil
	case cmd.Flag("user") != nil && cmd.Flag("user").Changed:
		ko, err := configfile.ReadConfigWithDefaults(rt.UserConfigFile)
		if err != nil {
			return nil, Errorf(CodeConfig, "failed to read user config file: %w", err)
		}
		return ko, nil
	case cmd.Flag("config") != nil && cmd.Flag("config").Changed:
		path := cmd.Flag("config").Value.String()
		ko, err := configfile.ReadConfigWithDefaults(path)
		if err != nil {
			return nil, Errorf(CodeConfig, "failed to read config file %s: %w", path, err)
		}
		return ko, nil
	default:
		return rt.Koanf, nil
	}
}
