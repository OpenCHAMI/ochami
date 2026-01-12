// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package service

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// serviceCmd represents the "bss service" command
	var serviceCmd = &cobra.Command{
		Use:   "service",
		Args:  cobra.NoArgs,
		Short: "Manage and check BSS itself",
		Long: `Manage and check BSS itself. This is a metacommand.

See ochami-bss(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
				os.Exit(0)
			}
		},
	}

	// Add subcommands
	serviceCmd.AddCommand(
		newCmdServiceStatus(),
		newCmdServiceVersion(),
	)

	return serviceCmd
}
