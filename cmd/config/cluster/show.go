// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cluster

import (
	"fmt"

	"github.com/knadh/koanf/v2"
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/config"
)

func newCmdClusterShow() *cobra.Command {
	// clusterShow represents the "config cluster show" command
	var clusterShowCmd = &cobra.Command{
		Use:   "show [cluster_name] [key]",
		Args:  cobra.MaximumNArgs(2),
		Short: "View cluster configuration options the CLI sees from a config file",
		Long: `View cluster configuration options the CLI sees from a config file.

See ochami-config(1) for details on the config commands.
See ochami-config(5) for details on the configuration options.`,
		Example: `  ochami config cluster show
  ochami config cluster show foobar
  ochami config cluster show foobar cluster.uri`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// It doesn't make sense to show the config of a config file
			// that doesn't exist, so err if the specified config file
			// doesn't exist.
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
			// Get the config from the relevant file depending on the flag,
			// or the merged config if none.
			var ko *koanf.Koanf
			var err error
			if cmd.Flags().Changed("system") {
				ko, err = config.ReadConfigWithDefaults(config.SystemConfigFile)
				if err != nil {
					return cli.Errorf(cli.CodeConfig, "failed to read system config file: %w", err)
				}
			} else if cmd.Flags().Changed("user") {
				ko, err = config.ReadConfigWithDefaults(config.UserConfigFile)
				if err != nil {
					return cli.Errorf(cli.CodeConfig, "failed to read user config file: %w", err)
				}
			} else if cmd.Flags().Changed("config") {
				ko, err = config.ReadConfigWithDefaults(cmd.Flag("config").Value.String())
				if err != nil {
					return cli.Errorf(cli.CodeConfig, "failed to read config file %s: %w", cmd.Flag("config").Value.String(), err)
				}
			} else {
				ko = config.GlobalKoanf
			}

			var key string
			var val string
			if len(args) == 0 {
				// No cluster specified, get all of them.
				val, err = config.GetConfigString(ko, "clusters")
				if err != nil {
					return cli.Errorf(cli.CodeConfig, "failed to fetch config for all clusters: %w", err)
				}
			} else {
				var cfgCl *config.ConfigCluster
				var clusters []config.ConfigCluster
				if err := ko.Unmarshal("clusters", &clusters); err != nil {
					return cli.Errorf(cli.CodeConfig, "failed to unmarshal clusters: %w", err)
				}
				for cidx, cl := range clusters {
					if cl.Name == args[0] {
						cfgCl = &(clusters[cidx])
						break
					}
				}
				if cfgCl == nil {
					return cli.Errorf(cli.CodeConfig, "cluster %q not found", args[0])
				}

				// Individual key was requested, print value directly
				if len(args) == 2 {
					key = args[1]
				}
				val, err = config.GetConfigClusterString(*cfgCl, key)
				if err != nil {
					if key == "" {
						return cli.Errorf(cli.CodeConfig, "failed to get full cluster config: %w", err)
					}
					return cli.Errorf(cli.CodeConfig, "failed to get cluster config for key %q: %w", key, err)
				}
			}
			if val != "" {
				fmt.Print(val)
			}

			return nil
		},
	}

	return clusterShowCmd
}
