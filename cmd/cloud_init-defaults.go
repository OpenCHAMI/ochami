// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// cloudInitDefaultsCmd represents the "cloud-init defaults" command
var cloudInitDefaultsCmd = &cobra.Command{
	Use:   "defaults",
	Args:  cobra.NoArgs,
	Short: "View cloud-init default values for the cluster",
	Long: `View default meta-data values for a cluster. This is a metacommand.
Commands under this one interact with the cloud-init
cluster-defaults endpoint.

See ochami-cloud-init(1) for more details.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		}
	},
}

func init() {
	cloudInitCmd.AddCommand(cloudInitDefaultsCmd)
}
