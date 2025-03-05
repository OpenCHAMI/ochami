// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/OpenCHAMI/ochami/internal/config"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/spf13/cobra"
)

// configClusterUnsetCmd represents the config-cluster-unset command
var configClusterUnsetCmd = &cobra.Command{
	Use:     "unset [--user | --system | --config <path>] <cluster_name> <key>",
	Short:   "Unset parameter for a cluster",
	Example: `  ochami config cluster unset foobar cluster.smd-uri`,
	PreRun: func(cmd *cobra.Command, args []string) {
		// To mark both persistent and regular flags mutually exclusive,
		// this function must be run before the command is executed. It
		// will not work in init(). This means that this needs to be
		// presend in all child commands.
		cmd.MarkFlagsMutuallyExclusive("system", "user", "config")
	},
	Run: func(cmd *cobra.Command, args []string) {
		// First and foremost, make sure config is loaded and logging
		// works.
		initConfigAndLogging(cmd, true)

		// Check that cluster name and key are only args
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		} else if len(args) != 2 {
			log.Logger.Error().Msgf("expected 2 arguments (cluster name, key) but got %d: %v", len(args), args)
			os.Exit(1)
		}

		// We must have a config file in order to write cluster info
		var fileToModify string
		if cmd.Flags().Changed("config") {
			fileToModify = configFile
		} else if configCmd.Flags().Changed("system") {
			fileToModify = config.SystemConfigFile
		} else {
			fileToModify = config.UserConfigFile
		}

		// Perform modification
		if err := config.DeleteConfigCluster(fileToModify, args[0], args[1]); err != nil {
			log.Logger.Error().Err(err).Msg("failed to modify config file")
			os.Exit(1)
		}
	},
}

func init() {
	configClusterCmd.AddCommand(configClusterUnsetCmd)
}
