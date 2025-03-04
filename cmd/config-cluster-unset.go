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
	Args:    cobra.ExactArgs(2),
	Short:   "Unset parameter for a cluster",
	Example: `  ochami config cluster unset foobar cluster.smd.uri`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// It doesn't make sense to unset a cluster config from a
		// non-existent config file, so err if the specified config
		// file doesn't exist.
		initConfigAndLogging(cmd, false)

		return nil
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// To mark both persistent and regular flags mutually exclusive,
		// this function must be run before the command is executed. It
		// will not work in init(). This means that this needs to be
		// presend in all child commands.
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
