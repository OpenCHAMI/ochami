// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rfe

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// rfeCmd represents the "smd rfe" command
	var rfeCmd = &cobra.Command{
		Use:   "rfe",
		Args:  cobra.NoArgs,
		Short: "Manage redfish endpoints",
		Long: `Manage redfish endpoints. This is a metacommand. Commands under this one
interact with the State Management Database (SMD).

See ochami-smd(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
				os.Exit(0)
			}
		},
	}

	// Add subcommands
	rfeCmd.AddCommand(
		newCmdRfeAdd(),
		newCmdRfeDelete(),
		newCmdRfeGet(),
	)

	return rfeCmd
}
