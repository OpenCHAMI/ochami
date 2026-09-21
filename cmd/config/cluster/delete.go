// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cluster

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/configfile"
	"github.com/openchami/ochami/internal/log"
)

func newCmdClusterDelete() *cobra.Command {
	// clusterDeleteCmd represents the "config cluster delete" command
	var clusterDeleteCmd = &cobra.Command{
		Use:   "delete <cluster_name>",
		Args:  cobra.ExactArgs(1),
		Short: "Delete a cluster from the configuration file",
		Long: `Delete a cluster from the configuration file.

See ochami-config(1) for details on the config commands.
See ochami-config(5) for details on configuration options.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// It doesn't make sense to delete a cluster from a
			// non-existent config file, so err if the config file doesn't
			// exist.
			return cli.InitConfigAndLogging(cmd, false)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// To mark both persistent and regular flags mutually exclusive,
			// this function must be run before the command is executed. It
			// will not work in init(). This means that this needs to be
			// present in all child commands.
			cmd.MarkFlagsMutuallyExclusive("system", "user", "config")

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// We must have a config file in order to write cluster info
			fileToModify := cli.ConfigFileToModify(cmd)
			clusterName := args[0]

			if err := configfile.DeleteCluster(fileToModify, clusterName); err != nil {
				return cli.Errorf(cli.CodeConfig, "failed to delete cluster %s from config file %s: %w", clusterName, fileToModify, err)
			}

			log.Logger.Info().Msgf("deleted cluster %s from config file %s", clusterName, fileToModify)

			return nil
		},
	}

	return clusterDeleteCmd
}
