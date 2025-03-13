// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// bootScriptCmd represents the bss-boot-script command
var bootScriptCmd = &cobra.Command{
	Use:   "script",
	Args:  cobra.NoArgs,
	Short: "Work with boot scripts for components",
	Long: `Work with boot scripts for components. This is a metacommand.

See ochami-bss(1) for more details.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		}
	},
}

func init() {
	bootCmd.AddCommand(bootScriptCmd)
}
