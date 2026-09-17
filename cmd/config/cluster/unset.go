// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cluster

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/config"
)

func newCmdClusterUnset() *cobra.Command {
	// clusterUnsetCmd represents the "config cluster unset" command
	var clusterUnsetCmd = &cobra.Command{
		Use:   "unset [--user | --system | --config <path>] <cluster_name> <key>",
		Args:  cobra.ExactArgs(2),
		Short: "Unset parameter for a cluster",
		Long: `Unset parameter for a cluster.

See ochami-config(1) for details on the config commands.
See ochami-config(5) for details on the configuration options.`,
		Example: `  ochami config cluster unset foobar cluster.smd.uri`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// It doesn't make sense to unset a cluster config from a
			// non-existent config file, so err if the specified config
			// file doesn't exist.
			return cli.InitConfigAndLogging(cmd, false)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// To mark both persistent and regular flags mutually exclusive,
			// this function must be run before the command is executed. It
			// will not work in init(). This means that this needs to be
			// presend in all child commands.
			cmd.MarkFlagsMutuallyExclusive("system", "user", "config")

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// We must have a config file in order to write cluster info
			var fileToModify string
			if cmd.Flags().Changed("config") {
				fileToModify = cli.ConfigFile
			} else if cmd.Parent().Parent().Flags().Changed("system") {
				// Check if --system was passed to 'config' command
				fileToModify = config.SystemConfigFile
			} else {
				fileToModify = config.UserConfigFile
			}

			// Perform modification
			if err := config.DeleteConfigCluster(fileToModify, args[0], args[1]); err != nil {
				return cli.Errorf(cli.CodeConfig, "failed to modify config file: %w", err)
			}

			return nil
		},
	}

	return clusterUnsetCmd
}
