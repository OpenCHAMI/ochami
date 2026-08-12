// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cluster

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/config"
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
			cli.InitConfigAndLogging(cmd, false)

			return nil
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// To mark both persistent and regular flags mutually exclusive,
			// this function must be run before the command is executed. It
			// will not work in init(). This means that this needs to be
			// present in all child commands.
			cmd.MarkFlagsMutuallyExclusive("system", "user", "config")

			// First and foremost, make sure config is loaded and logging
			// works.
			cli.InitConfigAndLogging(cmd, true)

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			// Get root command
			rootCmd := cmd.Root()
			_ = rootCmd // read persistent flags, annotations, etc.

			// We must have a config file in order to write cluster info
			var fileToModify string
			if rootCmd.PersistentFlags().Lookup("config").Changed {
				var err error
				if fileToModify, err = rootCmd.PersistentFlags().GetString("config"); err != nil {
					log.Logger.Error().Err(err).Msgf("unable to get value from --config flag")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}
			} else if cmd.Parent().Parent().PersistentFlags().Lookup("system").Changed {
				// Check if --system was passed to the 'config' command
				fileToModify = config.SystemConfigFile
			} else {
				fileToModify = config.UserConfigFile
			}

			// Read in config from file
			ko, err := config.ReadConfig(fileToModify)
			if err != nil {
				log.Logger.Error().Err(err).Msgf("failed to read config from %s", fileToModify)
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			var clusters []map[string]any
			err = ko.Unmarshal("clusters", &clusters)
			if err != nil {
				log.Logger.Error().Err(err).Msgf("unable to unmarshal clusters")
			}

			found := false
			clusterName := args[0]
			newClusters := make([]map[string]any, 0, len(clusters))
			for _, c := range clusters {
				if c["name"] != clusterName {
					newClusters = append(newClusters, c)
					found = true
				}
			}

			if !found {
				log.Logger.Error().Msgf("cluster '%s' doesn't exist", clusterName)
			}

			ko.Set("clusters", newClusters)

			// If we have reached here, the cluster was not found
			log.Logger.Error().Msgf("cluster %s not found in config file %s", clusterName, cli.ConfigFile)
			cli.LogHelpError(cmd)
			os.Exit(1)
		},
	}

	return clusterDeleteCmd
}
