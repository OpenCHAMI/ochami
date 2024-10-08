// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/synackd/ochami/internal/log"
)

// The 'config' command is a metacommand that allows the user to show and set
// configuration options in the passed config file.
var configCmd = &cobra.Command{
	Use:     "config",
	Args:    cobra.NoArgs,
	Short:   "Set or view configuration options",
	Example: `ochami config show`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			err := cmd.Usage()
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to print usage")
				os.Exit(1)
			}
			os.Exit(0)
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
