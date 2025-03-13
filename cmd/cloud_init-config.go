// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// cloudInitConfigCmd represents the cloud-init-config command
var cloudInitConfigCmd = &cobra.Command{
	Use:   "config",
	Args:  cobra.NoArgs,
	Short: "Manage cloud-init configurations for components",
	Long: `Manage cloud-init configurations for components. This is a metacommand. Commands
under this one interact with the cloud-init service and deal with
cloud-init configurations.

See ochami-cloud-init(1) for more details.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		}
	},
}

func init() {
	cloudInitCmd.AddCommand(cloudInitConfigCmd)
}
