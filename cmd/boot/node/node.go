// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package node

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// bootNodeCmd represents the "boot node" command
	var bootNodeCmd = &cobra.Command{
		Use:   "node",
		Args:  cobra.NoArgs,
		Short: "Manage nodes",
		Long: `Manage nodes known to boot service.

See ochami-boot(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cli.PrintUsageHandleError(cmd)
			}
			return nil
		},
	}

	// Create flags
	bootNodeCmd.PersistentFlags().BoolP("envelope", "e", false, "use the envelope (advanced) API, preserving metadata/labels/annotations, instead of the simple API")

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
