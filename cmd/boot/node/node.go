// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package node

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// bootNodeCmd represents the "boot node" command
	var bootNodeCmd = &cobra.Command{
		Use:   "node",
		Args:  cobra.NoArgs,
		Short: "Manage nodes",
		Long: `Manage nodes known to boot service.

See ochami-boot(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
				os.Exit(0)
			}
		},
	}

	// Add subcommands
	bootNodeCmd.AddCommand(
		newCmdBootNodeAdd(),
		newCmdBootNodeDelete(),
		newCmdBootNodeGet(),
		newCmdBootNodeList(),
		newCmdBootNodePatch(),
		newCmdBootNodeSet(),
	)

	return bootNodeCmd
}
