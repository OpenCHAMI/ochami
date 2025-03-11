// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// cloudInitCmd represents the cloud-init command
var cloudInitCmd = &cobra.Command{
	Use:   "cloud-init",
	Args:  cobra.NoArgs,
	Short: "Interact with the cloud-init service",
	Long: `Interact with the cloud-init service. This is a metacommand.

See ochami-cloud-init(1) for more details.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		}
	},
}

func init() {
	cloudInitCmd.PersistentFlags().String("uri", "", "absolute base URI or relative base path of cloud-init")
	rootCmd.AddCommand(cloudInitCmd)
}
