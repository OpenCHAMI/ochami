// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"strings"

	"github.com/OpenCHAMI/ochami/internal/config"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/spf13/cobra"
)

// The 'show' subcommand of the 'config' command prints out the configuration
// values that the CLI sees, whether that be from a flag
var configShowCmd = &cobra.Command{
	Use:   "show",
	Args:  cobra.NoArgs,
	Short: "View configuration options the CLI sees from a config file",
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
		switch format {
		case "yaml":
			cfgDataBytes, err = yaml.Marshal(config.GlobalConfig)
		case "json":
			cfgDataBytes, err = json.MarshalIndent(config.GlobalConfig, "", "\t")
		default:
			log.Logger.Error().Msgf("unknown log output format: %s", format)
			os.Exit(1)
		}
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to unmarshal configuration data")
			os.Exit(1)
		}
		fmt.Println(string(cfgDataBytes))
	},
}

func init() {
	configShowCmd.Flags().StringP("format", "f", "yaml", "format of config output (yaml,json)")
	configCmd.AddCommand(configShowCmd)
}
