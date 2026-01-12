// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package transition

import (
	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// transitionCmd represents the "pcs transitions" command
	var transitionCmd = &cobra.Command{
		Use:   "transition",
		Args:  cobra.NoArgs,
		Short: "Manage PCS transitions",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
			}
		},
	}

	// Add subcommands
	transitionCmd.AddCommand(
		newCmdTransitionAbort(),
		newCmdTransitionList(),
		newCmdTransitionMonitor(),
		newCmdTransitionShow(),
		newCmdTransitionStart(),
	)

	return transitionCmd
}
