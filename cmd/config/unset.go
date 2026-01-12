// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/config"
	"github.com/OpenCHAMI/ochami/internal/log"
)

func newCmdUnset() *cobra.Command {
	// unsetCmd represents the "config unset" command
	var unsetCmd = &cobra.Command{
		Use:   "unset [--user | --system | --config <path>] <key>",
		Args:  cobra.ExactArgs(1),
		Short: "Unset a key in ochami CLI configuration",
		Long: `Unset a key in ochami CLI configuration. By default, this command modifies
the user config file, which also occurs if --user is passed. If --system
is passed, this command edits the system configuration file. If --config
is passed instead, this command edits the file at the path specified.

This command does not handle cluster configs. For that, use the
'ochami config cluster delete' command.

See ochami-config(1) for details on the config commands.
See ochami-config(5) for details on the configuration options.`,
		Example: `  ochami config unset log.format
  ochami config unset --user log.format
  ochami config unset --system log.format
  ochami --config ./test.yaml config unset log.format`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// It doesn't make sense to unset from a config file that
			// doesn't exist, so err if the specified config file doesn't
			// exist.
			cli.InitConfigAndLogging(cmd, false)

			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// To mark both persistent and regular flags mutually exclusive,
			// this function must be run before the command is executed. It
			// will not work in init(). This means that this needs to be
			// present in all child commands.
			cmd.MarkFlagsMutuallyExclusive("system", "user", "config")

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			// Get root command
			rootCmd := cmd.Root()
			_ = rootCmd // read persistent flags, annotations, etc.

			// We must have a config file in order to write config
			var fileToModify string
			if rootCmd.Flags().Changed("config") {
				fileToModify = cli.ConfigFile
			} else if cmd.Parent().PersistentFlags().Lookup("system").Changed {
				// Check if --system was passed to 'config' command
				fileToModify = config.SystemConfigFile
			} else {
				fileToModify = config.UserConfigFile
			}

			// Refuse to modify config if user tries to modify cluster config
			if strings.HasPrefix(args[0], "clusters") {
				log.Logger.Error().Msg("`ochami config unset` is meant for unsetting general config, use `ochami config cluster delete` for deleting cluster config")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Perform modification
			if err := config.DeleteConfig(fileToModify, args[0]); err != nil {
				log.Logger.Error().Err(err).Msg("failed to modify config file")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
		},
	}

	return unsetCmd
}
