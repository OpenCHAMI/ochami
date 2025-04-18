// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// configClusterCmd represents the config-cluster command
var configClusterCmd = &cobra.Command{
	Use:   "cluster",
	Args:  cobra.NoArgs,
	Short: "Manage cluster configuration",
	Long: `Manage cluster configuration.

See ochami-config(1) for details on the config commands.
See ochami-config(5) for details on the configuration options.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// To mark both persistent and regular flags mutually exclusive,
		// this function must be run before the command is executed. It
		// will not work in init(). This means that this needs to be
		// present in all child commands.
		cmd.MarkFlagsMutuallyExclusive("system", "user", "config")

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		}
	},
}

func init() {
	configCmd.AddCommand(configClusterCmd)
}
