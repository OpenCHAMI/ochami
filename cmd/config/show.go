// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"strings"

	"github.com/knadh/koanf/v2"
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/configfile"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/config"
)

func newCmdShow() *cobra.Command {
	// show represents the "config show" command
	var showCmd = &cobra.Command{
		Use:   "show [key]",
		Args:  cobra.MaximumNArgs(1),
		Short: "View configuration options the CLI sees from a config file",
		Long: `View configuration options the CLI sees from a config file.

See ochami-config(1) for details on the config commands.
See ochami-config(5) for details on the configuration options.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			log.Logger.Debug().Msgf("COMMAND: %v", strings.Split(cmd.CommandPath(), " "))
			// To mark both persistent and regular flags mutually exclusive,
			// this function must be run before the command is executed. It
			// will not work in init(). This means that this needs to be
			// present in all child commands.
			cmd.MarkFlagsMutuallyExclusive("system", "user", "config")

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Get the config from the relevant file depending on the flag,
			// or the merged config if none.
			var ko *koanf.Koanf
			if cmd.Flags().Changed("system") {
				ko, err = configfile.ReadConfigWithDefaults(config.SystemConfigFile)
				if err != nil {
					return cli.Errorf(cli.CodeConfig, "failed to read system config file: %w", err)
				}
			} else if cmd.Flags().Changed("user") {
				ko, err = configfile.ReadConfigWithDefaults(rt.UserConfigFile)
				if err != nil {
					return cli.Errorf(cli.CodeConfig, "failed to read user config file: %w", err)
				}
			} else if cmd.Flags().Changed("config") {
				ko, err = configfile.ReadConfigWithDefaults(cmd.Flag("config").Value.String())
				if err != nil {
					return cli.Errorf(cli.CodeConfig, "failed to read config file %s: %w", cmd.Flag("config").Value.String(), err)
				}
			} else {
				ko = rt.Koanf
			}

			// Individual key was requested, print value directly
			var key string
			var val string
			if len(args) == 1 {
				key = args[0]
			}
			val, err = configfile.GetConfigString(ko, key)
			if err != nil {
				if key == "" {
					return cli.Errorf(cli.CodeConfig, "failed to get full config: %w", err)
				}
				return cli.Errorf(cli.CodeConfig, "failed to get config for key %q: %w", key, err)
			}
			if val != "" {
				if err := cli.WriteString(rt.Ios.Out(), val); err != nil {
					return err
				}
			}

			return nil
		},
	}

	return showCmd
}
