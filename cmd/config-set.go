// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/config"
	"github.com/OpenCHAMI/ochami/internal/log"
)

// configSetCmd represents the config-set command
var configSetCmd = &cobra.Command{
	Use:   "set [--user | --system | --config <path>] <key> <value>",
	Args:  cobra.ExactArgs(2),
	Short: "Modify ochami CLI configuration",
	Long: `Modify ochami CLI configuration. By default, this command modifies the user
config file, which also occurs if --user is passed. If --system is passed,
this command edits the system configuration file. If --config is passed
instead, this command edits the file at the path specified.

This command does not handle cluster configs. For that, use the
'ochami config cluster set' command.

See ochami-config(1) for details on the config commands.
See ochami-config(5) for details on the configuration options.`,
	Example: `  ochami config set log.format json
  ochami config set --user log.format json
  ochami config set --system log.format json
  ochami --config ./test.yaml config set log.format json`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// To mark both persistent and regular flags mutually exclusive,
		// this function must be run before the command is executed. It
		// will not work in init(). This means that this needs to be
		// present in all child commands.
		cmd.MarkFlagsMutuallyExclusive("system", "user", "config")

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// We must have a config file in order to write config
		var fileToModify string
		if cmd.Flags().Changed("config") {
			fileToModify = configFile
		} else if configCmd.PersistentFlags().Lookup("system").Changed {
			fileToModify = config.SystemConfigFile
		} else {
			fileToModify = config.UserConfigFile
		}

		// Refuse to modify config if user tries to modify cluster config
		if strings.HasPrefix(args[0], "clusters") {
			log.Logger.Error().Msg("`ochami config set` is meant for modifying general config, use `ochami config cluster set` for modifying cluster config")
			logHelpError(cmd)
			os.Exit(1)
		}

		// Handle file creation based on --no-confirm flag
		handleFileCreation(cmd, fileToModify)

		// Perform modification
		if err := config.ModifyConfig(fileToModify, args[0], args[1]); err != nil {
			log.Logger.Error().Err(err).Msg("failed to modify config file")
			logHelpError(cmd)
			os.Exit(1)
		}
	},
}

func init() {
	configCmd.AddCommand(configSetCmd)
}
