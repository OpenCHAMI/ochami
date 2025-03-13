// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rfeCmd represents the smd-rfe command
var rfeCmd = &cobra.Command{
	Use:   "rfe",
	Args:  cobra.NoArgs,
	Short: "Manage redfish endpoints",
	Long: `Manage redfish endpoints. This is a metacommand. Commands under this one
interact with the State Management Database (SMD).

See ochami-smd(1) for more details.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		}
	},
}

func init() {
	smdCmd.AddCommand(rfeCmd)
}
