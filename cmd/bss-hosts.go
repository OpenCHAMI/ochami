// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// bssHostsCmd represents the hosts command
var bssHostsCmd = &cobra.Command{
	Use:   "hosts",
	Args:  cobra.NoArgs,
	Short: "Work with hosts in BSS",
	Long: `Work with hosts in BSS.

See ochami-bss(1) for more details.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		}
	},
}

func init() {
	bssCmd.AddCommand(bssHostsCmd)
}
