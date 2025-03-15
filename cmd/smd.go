// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// smdCmd represents the bss command
var smdCmd = &cobra.Command{
	Use:   "smd",
	Args:  cobra.NoArgs,
	Short: "Communicate with the State Management Database (SMD)",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		}
	},
}

func init() {
	rootCmd.AddCommand(smdCmd)
}
