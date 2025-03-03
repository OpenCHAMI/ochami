// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/spf13/cobra"
)

// bssCmd represents the bss command
var bssCmd = &cobra.Command{
	Use:   "bss",
	Args:  cobra.NoArgs,
	Short: "Communicate with the Boot Script Service (BSS)",
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
	bssCmd.PersistentFlags().String("uri", "", "absolute base URI or relative base path of BSS")
	rootCmd.AddCommand(bssCmd)
}
