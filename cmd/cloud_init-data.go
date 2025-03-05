// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// cloudInitDataCmd represents the cloud-init-data command
var cloudInitDataCmd = &cobra.Command{
	Use:   "data",
	Args:  cobra.NoArgs,
	Short: "View cloud-init data",
	Long: `View cloud-init data. This is a metacommand. Commands under this one
interact with the cloud-init service and deal with cloud-init data.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		}
	},
}

func init() {
	cloudInitCmd.AddCommand(cloudInitDataCmd)
}
