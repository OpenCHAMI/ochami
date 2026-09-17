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
	"github.com/openchami/ochami/pkg/config"
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

			// First and foremost, make sure config is loaded and logging
			// works.
			return cli.InitConfigAndLogging(cmd, true)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get root command
			rootCmd := cmd.Root()
			_ = rootCmd // read persistent flags, annotations, etc.

			// We must have a config file in order to write cluster info
			var fileToModify string
			if rootCmd.PersistentFlags().Lookup("config").Changed {
				var err error
				if fileToModify, err = rootCmd.PersistentFlags().GetString("config"); err != nil {
					return cli.Errorf(cli.CodeUsage, "unable to get value from --config flag: %w", err)
				}
			} else if cmd.Parent().Parent().PersistentFlags().Lookup("system").Changed {
				// Check if --system was passed to the 'config' command
				fileToModify = config.SystemConfigFile
			} else {
				fileToModify = cli.UserConfigFile
			}

			// Read in config from file
			ko, err := configfile.ReadConfig(fileToModify)
			if err != nil {
				return cli.Errorf(cli.CodeConfig, "failed to read config from %s: %w", fileToModify, err)
			}

			var clusters []map[string]any
			if err := ko.Unmarshal("clusters", &clusters); err != nil {
				return cli.Errorf(cli.CodeConfig, "unable to unmarshal clusters: %w", err)
			}

			found := false
			clusterName := args[0]
			newClusters := make([]map[string]any, 0, len(clusters))
			for _, c := range clusters {
				if c["name"] == clusterName {
					found = true
					continue
				}
				newClusters = append(newClusters, c)
			}

			// It doesn't make sense to delete a cluster that doesn't
			// exist, so err before writing anything back to the file.
			if !found {
				return cli.Errorf(cli.CodeConfig, "cluster %s not found in config file %s", clusterName, fileToModify)
			}

			if err := ko.Set("clusters", newClusters); err != nil {
				return cli.Errorf(cli.CodeConfig, "failed to set clusters: %w", err)
			}

			if clusterName == ko.String("default-cluster") {
				ko.Delete("default-cluster")
			}

			// Write config to file
			if err := configfile.WriteConfig(fileToModify, ko); err != nil {
				return cli.Errorf(cli.CodeConfig, "failed to write config to %s: %w", fileToModify, err)
			}

			log.Logger.Info().Msgf("deleted cluster %s from config file %s", clusterName, fileToModify)

			return nil
		},
	}

	return clusterDeleteCmd
}
