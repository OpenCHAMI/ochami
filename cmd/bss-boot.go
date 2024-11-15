// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/OpenCHAMI/ochami/internal/log"
)

// bootCmd represents the bss-boot command
var bootCmd = &cobra.Command{
	Use:   "boot",
	Args:  cobra.NoArgs,
	Short: "Manage boot configuration for components",
	Long: `Manage boot configuration for components. This is a metacommand. Commands
under this one interact with the Boot Script Service (BSS).`,
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
	bssCmd.AddCommand(bootCmd)
}
