// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"github.com/spf13/cobra"
)

// pcsStatusCmd represents the "pcs status" command
var pcsStatusCmd = &cobra.Command{
	Use:   "status",
	Args:  cobra.NoArgs,
	Short: "Manage PCS status",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			printUsageHandleError(cmd)
		}
	},
}

func init() {
	pcsCmd.AddCommand(pcsStatusCmd)
}
