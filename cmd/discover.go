// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// discoverCmd represents the discover command
var discoverCmd = &cobra.Command{
	Use:   "discover",
	Args:  cobra.NoArgs,
	Short: "Perform static or dynamic discovery of nodes",
	Run: func(cmd *cobra.Command, args []string) {
		// Check that all required args are passed
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		}
	},
}

func init() {
	rootCmd.AddCommand(discoverCmd)
}
