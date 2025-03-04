// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"os"
	"strings"

	"github.com/OpenCHAMI/ochami/internal/config"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/spf13/cobra"
)

// configSetCmd represents the config-set command
var configSetCmd = &cobra.Command{
	Use:   "set [--user | --system | --config <path>] <key> <value>",
	Short: "Modify ochami CLI configuration",
	Long: `Modify ochami CLI configuration. By default, this command modifies the user
config file, which also occurs if --user is passed. If --system is passed,
this command edits the system configuration file. If --config is passed
instead, this command edits the file at the path specified.

This command does not handle cluster configs. For that, use the
'ochami config cluster set' command.`,
	Example: `  ochami config set log.format json
  ochami config set --user log.format json
  ochami config set --system log.format json
  ochami --config ./test.yaml config set log.format json`,
	Run: func(cmd *cobra.Command, args []string) {
		// Ensure we have 2 args
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		} else if len(args) != 2 {
			log.Logger.Error().Msgf("expected 2 arguments (key, value) but got %s: %v", len(args), args)
			os.Exit(1)
		}

		// We must have a config file in order to write config
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

		// Refuse to modify config if user tries to modify cluster config
		if strings.HasPrefix(args[0], "clusters") {
			log.Logger.Error().Msg("`ochami config set` is meant for modifying general config, use `ochami config cluster set` for modifying cluster config")
			os.Exit(1)
		}

		// Perform modification
		if err := config.ModifyConfig(fileToModify, args[0], args[1]); err != nil {
			log.Logger.Error().Err(err).Msg("failed to modify config file")
			os.Exit(1)
		}
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
}
