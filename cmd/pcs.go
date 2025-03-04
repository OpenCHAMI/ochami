// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// pcsCmd represents the pcs command
var pcsCmd = &cobra.Command{
	Use:   "pcs",
	Args:  cobra.NoArgs,
	Short: "Interact with the Power Control Service (PCS)",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		}
	},
}

func init() {
	pcsCmd.PersistentFlags().String("uri", "", "absolute base URI or relative base path of PCS")
	rootCmd.AddCommand(pcsCmd)
}
