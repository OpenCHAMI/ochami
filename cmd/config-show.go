// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/OpenCHAMI/ochami/internal/config"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/spf13/cobra"
)

// configShow represents the config-show command
var configShowCmd = &cobra.Command{
	Use:   "show [key]",
	Args:  cobra.MaximumNArgs(1),
	Short: "View configuration options the CLI sees from a config file",
	Long: `View configuration options the CLI sees from a config file.

See ochami-config(1) for details on the config commands.
See ochami-config(5) for details on the configuration options.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// It doesn't make sense to show the config value from a config
		// file that doesn't exist, so err if the specified config file
		// doesn't exist.
		initConfigAndLogging(cmd, false)

		return nil
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		log.Logger.Debug().Msgf("COMMAND: %v", strings.Split(cmd.CommandPath(), " "))
		// To mark both persistent and regular flags mutually exclusive,
		// this function must be run before the command is executed. It
		// will not work in init(). This means that this needs to be
		// present in all child commands.
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
				log.Logger.Error().Err(err).Msgf("failed to read user config file")
				logHelpError(cmd)
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

		// Individual key was requested, print value directly
		var key string
		var val string
		if len(args) == 1 {
			key = args[0]
		}
		val, err = config.GetConfigString(cfg, key, format)
		if err != nil {
			if key == "" {
				log.Logger.Error().Err(err).Msgf("failed to get full config")
			} else {
				log.Logger.Error().Err(err).Msgf("failed to get config for key %q", key)
			}
			logHelpError(cmd)
			os.Exit(1)
		}
		if val != "" {
			fmt.Printf("%v\n", val)
		}
	},
}

func init() {
	configShowCmd.Flags().StringP("format", "f", "yaml", "format of config output (yaml,json,json-pretty)")
	configCmd.AddCommand(configShowCmd)
}
