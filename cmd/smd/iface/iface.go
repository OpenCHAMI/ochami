// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package iface

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// ifaceCmd represents the "smd iface" command
	var ifaceCmd = &cobra.Command{
		Use:   "iface",
		Args:  cobra.NoArgs,
		Short: "Manage ethernet interfaces",
		Long: `Manage ethernet interfaces. This is a metacommand. Commands under this one
interact with the State Management Database (SMD).

See ochami-smd(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cli.PrintUsageHandleError(cmd)
			}
			return nil
		},
	}

	// Add subcommands
	ifaceCmd.AddCommand(
		newCmdIfaceAdd(),
		newCmdIfaceDelete(),
		newCmdIfaceGet(),
	)

	return ifaceCmd
}
