// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// bootParamsCmd represents the bss-boot-params command
var bootParamsCmd = &cobra.Command{
	Use:   "params",
	Args:  cobra.NoArgs,
	Short: "Work with boot parameters for components",
	Long: `Work with boot parameters for components, including kernel URI, initrd URI,
and kernel command line arguments. This is a metacommand.

See ochami-bss(1) for more details.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		}
	},
}

func init() {
	bootCmd.AddCommand(bootParamsCmd)
}
