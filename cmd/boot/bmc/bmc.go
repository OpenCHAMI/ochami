// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package bmc

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// bootBmcCmd represents the "boot bmc" command
	var bootBmcCmd = &cobra.Command{
		Use:   "bmc",
		Args:  cobra.NoArgs,
		Short: "Manage BMCs",
		Long: `Manage BMCs known to boot service.

See ochami-boot(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
				os.Exit(0)
			}
		},
	}

	// Add subcommands
	bootBmcCmd.AddCommand(
		newCmdBootBmcAdd(),
		newCmdBootBmcDelete(),
		newCmdBootBmcGet(),
		newCmdBootBmcList(),
		newCmdBootBmcPatch(),
		newCmdBootBmcSet(),
	)

	return bootBmcCmd
}
