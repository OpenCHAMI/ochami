// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// ifaceCmd represents the smd-iface command
var ifaceCmd = &cobra.Command{
	Use:   "iface",
	Args:  cobra.NoArgs,
	Short: "Manage ethernet interfaces",
	Long: `Manage ethernet interfaces. This is a metacommand. Commands under this one
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
	smdCmd.AddCommand(ifaceCmd)
}
