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
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cli.PrintUsageHandleError(cmd)
			}
			return nil
		},
	}

	// Add subcommands
	pcsStatusCmd.AddCommand(
		newCmdStatusList(),
		newCmdStatusShow(),
	)

	return pcsStatusCmd
}
