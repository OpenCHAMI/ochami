// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// cloudInitGroupCmd represents the "cloud-init group" command
var cloudInitGroupCmd = &cobra.Command{
	Use:   "group",
	Args:  cobra.NoArgs,
	Short: "Manage cloud-init groups",
	Long: `Manage cloud-init groups.

See ochami-cloud-init(1) for more details.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		}
	},
}

func init() {
	cloudInitCmd.AddCommand(cloudInitGroupCmd)
}
