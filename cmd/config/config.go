// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"

	// Subcommands
	cluster_cmd "github.com/OpenCHAMI/ochami/cmd/config/cluster"
)

func NewCmd() *cobra.Command {
	// The 'config' command is a metacommand that allows the user to show and set
	// configuration options in the passed config file.
	var configCmd = &cobra.Command{
		Use:   "config",
		Args:  cobra.NoArgs,
		Short: "Set or view configuration options",
		Long: `Set or view configuration options.

See ochami-config(1) for details on the config commands.
See ochami-config(5) for details on the configuration options.`,
		Example: `  ochami config show`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// To mark both persistent and regular flags mutually exclusive,
			// this function must be run before the command is executed. It
			// will not work in init(). This means that this needs to be
			// present in all child commands.
			cmd.MarkFlagsMutuallyExclusive("system", "user", "config")

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
				os.Exit(0)
			}
		},
	}

	// Create flags
	configCmd.PersistentFlags().Bool("system", false, "modify system config")
	configCmd.PersistentFlags().Bool("user", true, "modify user config")

	// Add subcommands
	configCmd.AddCommand(
		cluster_cmd.NewCmd(),
		newCmdSet(),
		newCmdShow(),
		newCmdUnset(),
	)

	return configCmd
}
