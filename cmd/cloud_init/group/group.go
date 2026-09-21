// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// groupCmd represents the "cloud-init group" command
	var groupCmd = &cobra.Command{
		Use:   "group",
		Args:  cobra.NoArgs,
		Short: "Manage cloud-init groups",
		Long: `Manage cloud-init groups.

See ochami-cloud-init(1) for more details.`,
		RunE: cli.PrintUsage,
	}

	// Add subcommands
	groupCmd.AddCommand(
		newCmdGroupAdd(),
		newCmdGroupDelete(),
		newCmdGroupGet(),
		newCmdGroupRender(),
		newCmdGroupSet(),
	)

	return groupCmd
}
