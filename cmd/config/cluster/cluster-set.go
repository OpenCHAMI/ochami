// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cluster

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/config"
	"github.com/OpenCHAMI/ochami/internal/log"
)

func newCmdClusterSet() *cobra.Command {
	// clusterSetCmd represents the "config cluster set" command
	var clusterSetCmd = &cobra.Command{
		Use:   "set [--user | --system | --config <path>] [-d] <cluster_name> <key> <value>",
		Args:  cobra.ExactArgs(3),
		Short: "Add or set parameters for a cluster",
		Long: `Add cluster with its configuration or set the configuration for
an existing cluster. For example:

	ochami config cluster set foobar cluster.uri https://foobar.openchami.cluster

Creates the following entry in the 'clusters' list:

	- name: foobar
	  cluster:
	    uri: https://foobar.openchami.cluster

If this is the first cluster created, the following is also set:

	default-cluster: foobar

default-cluster is used to determine which cluster in the list should be used for subcommands.

This same command can be use to modify existing cluster information. Running the same command above
with a different base URI will change the cluster base URI for the 'foobar' cluster.

See ochami-config(1) for details on the config commands.
See ochami-config(5) for details on the configuration options.`,
		Example: `  ochami config cluster set foobar cluster.uri https://foobar.openchami.cluster
  ochami config cluster set foobar cluster.smd.uri /hsm/v2
  ochami config cluster set foobar name new-foobar`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// To mark both persistent and regular flags mutually exclusive,
			// this function must be run before the command is executed. It
			// will not work in init(). This means that this needs to be
			// present in all child commands.
			cmd.MarkFlagsMutuallyExclusive("system", "user", "config")

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			// We must have a config file in order to write cluster info
			var fileToModify string
			if cmd.Flags().Changed("config") {
				fileToModify = cli.ConfigFile
			} else if cmd.Parent().Parent().Flags().Changed("system") {
				// Check if --system passed to 'config' command
				fileToModify = config.SystemConfigFile
			} else {
				fileToModify = config.UserConfigFile
			}

			// Ask to create file if it doesn't exist
			if create, err := cli.Ios.AskToCreate(fileToModify); err != nil {
				if err != cli.FileExistsError {
					log.Logger.Error().Err(err).Msg("error asking to create file")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}
			} else if create {
				if err := cli.CreateIfNotExists(fileToModify); err != nil {
					log.Logger.Error().Err(err).Msg("error creating file")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}
			} else {
				log.Logger.Error().Msg("user declined to create file, not modifying")
				os.Exit(0)
			}

			// Perform modification
			dflt, err := cmd.Flags().GetBool("default")
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to retrieve \"default\" flag")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
			if err := config.ModifyConfigCluster(fileToModify, args[0], args[1], dflt, config.StringToType(args[2])); err != nil {
				log.Logger.Error().Err(err).Msg("failed to modify config file")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
		},
	}

	// Create flags
	clusterSetCmd.Flags().BoolP("default", "d", false, "set cluster as the default")

	return clusterSetCmd
}
