// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// cloudInitNodeCmd represents the "cloud-init node" command
var cloudInitNodeCmd = &cobra.Command{
	Use:   "node",
	Args:  cobra.NoArgs,
	Short: "Manage cloud-init node-specific config",
	Long: `Manage cloud-init node-specific config.

See ochami-cloud-init(1) for more details.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		}
	},
}

func init() {
	cloudInitCmd.AddCommand(cloudInitNodeCmd)
}
