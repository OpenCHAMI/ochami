// SPDX-FileCopyrightText: © 2024-2026 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package status

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// pcsStatusCmd represents the "pcs status" command
	var pcsStatusCmd = &cobra.Command{
		Use:   "status",
		Args:  cobra.NoArgs,
		Short: "Manage PCS status",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
			}
		},
	}

	// Add subcommands
	pcsStatusCmd.AddCommand(
		newCmdStatusList(),
		newCmdStatusShow(),
	)

	return pcsStatusCmd
}
