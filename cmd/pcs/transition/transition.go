// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package transition

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// transitionCmd represents the "pcs transitions" command
	var transitionCmd = &cobra.Command{
		Use:   "transition",
		Args:  cobra.NoArgs,
		Short: "Manage PCS transitions",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cli.PrintUsageHandleError(cmd)
			}
			return nil
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
