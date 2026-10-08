// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cluster

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/configfile"
	"github.com/openchami/ochami/pkg/config"
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

Passing -d also makes it the default cluster:

	default-cluster: foobar

default-cluster is used to determine which cluster in the list should be used for subcommands.

This same command can be used to modify existing cluster information. Running the same command above
with a different base URI will change the cluster base URI for the 'foobar' cluster. Setting the
'name' key renames an existing cluster; it fails if the cluster does not exist. A cluster
name must be non-empty and must not contain a period ('.').

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
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// We must have a config file in order to write cluster info
			fileToModify := rt.ConfigFileToModify(cmd)

			// Ask to create file if it doesn't exist
			if create, err := rt.AskToCreate(fileToModify); err != nil {
				if !errors.Is(err, cli.ErrFileExists) {
					return cli.Errorf(cli.CodeConfig, "error asking to create file: %w", err)
				}
			} else if create {
				if err := rt.CreateIfNotExists(fileToModify); err != nil {
					return cli.Errorf(cli.CodeConfig, "error creating file: %w", err)
				}
			} else {
				return cli.Errorf(cli.CodeDeclined, "user declined to create file, not modifying")
			}

			// Perform modification
			dflt, err := cmd.Flags().GetBool("default")
			if err != nil {
				return cli.Errorf(cli.CodeUsage, "failed to retrieve \"default\" flag: %w", err)
			}

			f, err := config.OpenFile(fileToModify)
			if err != nil {
				return cli.Errorf(cli.CodeConfig, "failed to open config file: %w", err)
			}

			// If the key is "name", this is a rename. The new name is taken
			// as the raw argument rather than run through StringToType, since
			// a cluster name is always a string regardless of what it looks
			// like (e.g. a name of "true" or "42" should stay a string).
			//
			// Both the key change and the optional default-cluster update
			// (below) are made through a single Update call so they're
			// committed to disk in one write.
			targetName := args[0]
			err = f.Update(func(f *config.File) error {
				if args[1] == "name" {
					if err := f.RenameCluster(args[0], args[2]); err != nil {
						return err
					}
					targetName = args[2]
				} else if err := f.SetClusterKey(args[0], args[1], configfile.StringToType(args[2])); err != nil {
					return err
				}

				if dflt {
					if err := f.SetDefaultCluster(targetName); err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				return cli.Errorf(cli.CodeConfig, "failed to modify config file: %w", err)
			}

			return nil
		},
	}

	// Create flags
	clusterSetCmd.Flags().BoolP("default", "d", false, "set cluster as the default")

	return clusterSetCmd
}
