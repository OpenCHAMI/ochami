// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package member

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// groupMemberCmd represents the "smd group member" command
	var groupMemberCmd = &cobra.Command{
		Use:   "member",
		Args:  cobra.NoArgs,
		Short: "Manage group membership",
		Long: `Mange group membership. This is a metacommand. Commands under this one
interact with the State Management Database (SMD).

See ochami-smd(1) for more details.`,
		RunE: cli.PrintUsage,
	}

	// Add subcommands
	groupMemberCmd.AddCommand(
		newCmdGroupMemberAdd(),
		newCmdGroupMemberDelete(),
		newCmdGroupMemberGet(),
		newCmdGroupMemberSet(),
	)

	return groupMemberCmd
}
