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
	Use:   "set [--user | --system] <cluster_name>",
	Short: "Add or set parameters for a cluster",
	Long: `Add cluster with its configuration or set the configuration for
an existing cluster. For example:

	ochami config cluster set foobar --base-uri https://foobar.openchami.cluster

Creates the following entry in the 'clusters' list:

	- name: foobar
	  cluster:
	    base-uri: https://foobar.openchami.cluster

If this is the first cluster created, the following is also set:

	default-cluster: foobar

default-cluster is used to determine which cluster in the list should be used for subcommands.

This same command can be use to modify existing cluster information. Running the same command above
with a different base URL will change the base URL for the 'foobar' cluster.`,
	Example: `  ochami config cluster set foobar.openchami.cluster --base-uri https://foobar.openchami.cluster`,
	Run: func(cmd *cobra.Command, args []string) {
		// Check that cluster name is only arg
		if len(args) == 0 {
			err := cmd.Usage()
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to print usage")
				os.Exit(1)
			}
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

		// Read in config from file
		cfg, err := config.ReadConfig(fileToModify)
		if err != nil {
			log.Logger.Error().Err(err).Msgf("failed to read config from %s", fileToModify)
		}

		// Fetch existing cluster list config
		clusterName := args[0]
		clusterUrl := cmd.Flag("base-uri").Value.String()
		clusterIdx := -1

		// If cluster name already exists, we are modifying it instead of creating a new one
		for idx, cluster := range cfg.Clusters {
			if cluster.Name == clusterName {
				clusterIdx = idx
				break
			}
		}

		if clusterIdx == -1 {
			// Cluster does not exist, create a new entry for it in the config
			newCluster := config.ConfigCluster{
				Name: clusterName,
			}
			if clusterUrl != "" {
				newCluster.Cluster.APIURI = clusterUrl
				log.Logger.Debug().Msgf("using api-uri %s", clusterUrl)
			}

			// If this is the first cluster to be added, set it as the default
			if len(cfg.Clusters) == 0 {
				cfg.DefaultCluster = clusterName
				log.Logger.Info().Msgf("first and new cluster %s set as default-cluster", clusterName)
			}

			// Add new cluster to list
			cfg.Clusters = append(cfg.Clusters, newCluster)
			log.Logger.Info().Msgf("added new cluster: %s", clusterName)
		} else {
			// Cluster exists, modify it
			if clusterUrl != "" {
				cfg.Clusters[clusterIdx].Cluster.APIURI = clusterUrl
				log.Logger.Debug().Msgf("updating base-uri for cluster %s: %s", clusterName, clusterUrl)
			}
			log.Logger.Info().Msgf("modified config for existing cluster: %s", clusterName)
		}

		// If --default was passed, make this cluster the default one
		if cmd.Flag("default").Changed {
			cfg.DefaultCluster = clusterName
			log.Logger.Info().Msgf("cluster %s set as default-cluster since --default passed", clusterName)
		}

		// Write out modified config to the config file
		// WARNING: This will rewrite the whole config file so modifications like
		// comments will get erased.
		if err := config.WriteConfig(fileToModify, cfg); err != nil {
			log.Logger.Error().Err(err).Msgf("failed to write modified config to %s", fileToModify)
			os.Exit(1)
		}
	},
}

func init() {
	configClusterSetCmd.Flags().StringP("api-uri", "u", "", "base URL of cluster")
	configClusterSetCmd.Flags().BoolP("default", "d", false, "set cluster as the default")
	configClusterCmd.AddCommand(configClusterSetCmd)
}
