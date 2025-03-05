// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/OpenCHAMI/ochami/internal/config"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/spf13/cobra"
)

// configClusterDeleteCmd represents the config-cluster-delete command
var configClusterDeleteCmd = &cobra.Command{
	Use:   "delete <cluster_name>",
	Short: "Delete a cluster from the configuration file",
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
		} else if len(args) > 1 {
			log.Logger.Error().Msgf("expected 1 argument (cluster name) but got %d: %v", len(args), args)
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

		// Read in config from file
		cfg, err := config.ReadConfig(fileToModify)
		if err != nil {
			log.Logger.Error().Err(err).Msgf("failed to read config from %s", fileToModify)
		}

		// Fetch existing cluster list config
		clusterName := args[0]
		for idx, cluster := range cfg.Clusters {
			if cluster.Name == clusterName {
				cfg.Clusters = config.RemoveFromSlice(cfg.Clusters, idx)

				// If cluster was default, remove default-cluster
				if cfg.DefaultCluster != "" {
					if cfg.DefaultCluster == clusterName {
						cfg.DefaultCluster = ""
						log.Logger.Info().Msgf("cluster %s removed as default-cluster from config file %s", clusterName, fileToModify)
					}
				}

				// Write out config file
				// WARNING: This will rewrite the whole config file so modifications like
				// comments will get erased.
				if err := config.WriteConfig(fileToModify, cfg); err != nil {
					log.Logger.Error().Err(err).Msgf("failed to write modified config to %s", fileToModify)
					os.Exit(1)
				}
				log.Logger.Info().Msgf("cluster %s removed from config file %s", clusterName, configFile)

				os.Exit(0)
			}
		}

		// If we have reached here, the cluster was not found
		log.Logger.Error().Msgf("cluster %s not found in config file %s", clusterName, configFile)
		os.Exit(1)
	},
}

func init() {
	configClusterCmd.AddCommand(configClusterDeleteCmd)
}
