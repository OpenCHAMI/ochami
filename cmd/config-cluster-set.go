// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/OpenCHAMI/ochami/internal/config"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/spf13/cobra"
)

// configClusterSetCmd represents the config-cluster-set command
var configClusterSetCmd = &cobra.Command{
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
with a different base URI will change the cluster base URI for the 'foobar' cluster.`,
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
			fileToModify = configFile
		} else if configCmd.Flags().Changed("system") {
			fileToModify = config.SystemConfigFile
		} else {
			fileToModify = config.UserConfigFile
		}

		// Ask to create file if it doesn't exist
		if create, err := askToCreate(fileToModify); err != nil {
			if err != FileExistsError {
				log.Logger.Error().Err(err).Msg("error asking to create file")
				os.Exit(1)
			}
		} else if create {
			if err := createIfNotExists(fileToModify); err != nil {
				log.Logger.Error().Err(err).Msg("error creating file")
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
			os.Exit(1)
		}
		if err := config.ModifyConfigCluster(fileToModify, args[0], args[1], dflt, args[2]); err != nil {
			log.Logger.Error().Err(err).Msg("failed to modify config file")
			os.Exit(1)
		}
	},
}

func init() {
	configClusterSetCmd.Flags().BoolP("default", "d", false, "set cluster as the default")
	configClusterCmd.AddCommand(configClusterSetCmd)
}
