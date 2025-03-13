// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"fmt"
	"os"

	"github.com/OpenCHAMI/ochami/internal/config"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/spf13/cobra"
)

// configClusterShow represents the config-cluster-show command
var configClusterShowCmd = &cobra.Command{
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
		// Get the config from the relevant file depending on the flag,
		// or the merged config if none.
		var cfg config.Config
		var err error
		format := cmd.Flag("format").Value.String()
		if cmd.Flags().Changed("system") {
			cfg, err = config.ReadConfig(config.SystemConfigFile)
			if err != nil {
				log.Logger.Error().Err(err).Msgf("failed to read system config file")
				logHelpError(cmd)
				os.Exit(1)
			}
		} else if cmd.Flags().Changed("user") {
			cfg, err = config.ReadConfig(config.UserConfigFile)
			if err != nil {
				logHelpError(cmd)
				log.Logger.Error().Err(err).Msgf("failed to read user config file")
				os.Exit(1)
			}
		} else if cmd.Flags().Changed("config") {
			cfg, err = config.ReadConfig(cmd.Flag("config").Value.String())
			if err != nil {
				log.Logger.Error().Err(err).Msgf("failed to read config file %s", cmd.Flag("config").Value.String())
				logHelpError(cmd)
				os.Exit(1)
			}
		} else {
			cfg = config.GlobalConfig
		}

		var key string
		var val string
		if len(args) == 0 {
			// No cluster specified, get all of them.
			val, err = config.GetConfigString(cfg, "clusters", format)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to fetch config for all clusters")
				logHelpError(cmd)
				os.Exit(1)
			}
		} else {
			var cfgCl *config.ConfigCluster
			for cidx, cl := range cfg.Clusters {
				if cl.Name == args[0] {
					cfgCl = &(cfg.Clusters[cidx])
					break
				}
			}
			if cfgCl == nil {
				log.Logger.Error().Msgf("cluster %q not found", args[0])
				logHelpError(cmd)
				os.Exit(1)
			}

			// Individual key was requested, print value directly
			if len(args) == 2 {
				key = args[1]
			}
			val, err = config.GetConfigClusterString(*cfgCl, key, format)
			if err != nil {
				if key == "" {
					log.Logger.Error().Err(err).Msgf("failed to get full cluster config")
				} else {
					log.Logger.Error().Err(err).Msgf("failed to get cluster config for key %q", key)
				}
				logHelpError(cmd)
				os.Exit(1)
			}
		}
		if val != "" {
			fmt.Printf("%v\n", val)
		}
	},
}

func init() {
	configClusterShowCmd.Flags().StringP("format", "f", "yaml", "format of config output (yaml,json,json-pretty)")
	configClusterCmd.AddCommand(configClusterShowCmd)
}
