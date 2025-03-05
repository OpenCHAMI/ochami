// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"os"

	"github.com/OpenCHAMI/ochami/internal/config"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/spf13/cobra"
)

// configClusterSetCmd represents the config-cluster-set command
var configClusterSetCmd = &cobra.Command{
	Use:   "set [--user | --system | --config <path>] [-d] <cluster_name> <key> <value>",
	Short: "Add or set parameters for a cluster",
	Long: `Add cluster with its configuration or set the configuration for
an existing cluster. For example:

	ochami config cluster set foobar cluster.api-uri https://foobar.openchami.cluster

Creates the following entry in the 'clusters' list:

	- name: foobar
	  cluster:
	    base-uri: https://foobar.openchami.cluster

If this is the first cluster created, the following is also set:

	default-cluster: foobar

default-cluster is used to determine which cluster in the list should be used for subcommands.

This same command can be use to modify existing cluster information. Running the same command above
with a different base URL will change the API base URL for the 'foobar' cluster.`,
	Example: `  ochami config cluster set foobar cluster.api-uri https://foobar.openchami.cluster
  ochami config cluster set foobar cluster.smd-uri /hsm/v2
  ochami config cluster set foobar name new-foobar`,
	PreRun: func(cmd *cobra.Command, args []string) {
		// To mark both persistent and regular flags mutually exclusive,
		// this function must be run before the command is executed. It
		// will not work in init(). This means that this needs to be
		// presend in all child commands.
		cmd.MarkFlagsMutuallyExclusive("system", "user", "config")
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Check that cluster name is only arg
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		} else if len(args) != 3 {
			log.Logger.Error().Msgf("expected 3 arguments (cluster name, key, value) but got %d: %v", len(args), args)
			os.Exit(1)
		}

		// We must have a config file in order to write cluster info
		var fileToModify string
		if rootCmd.PersistentFlags().Lookup("config").Changed {
			var err error
			if fileToModify, err = rootCmd.PersistentFlags().GetString("config"); err != nil {
				log.Logger.Error().Err(err).Msgf("unable to get value from --config flag")
				os.Exit(1)
			}
		} else if configCmd.PersistentFlags().Lookup("system").Changed {
			fileToModify = config.SystemConfigFile
		} else {
			fileToModify = config.UserConfigFile
		}

		// Ask user to create file if it does not exist
		if err := askToCreate(fileToModify); err != nil {
			if errors.Is(err, UserDeclinedError) {
				log.Logger.Info().Msgf("user declined creating config file %s, exiting")
				os.Exit(0)
			} else {
				log.Logger.Error().Err(err).Msgf("failed to create %s")
				os.Exit(1)
			}
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
