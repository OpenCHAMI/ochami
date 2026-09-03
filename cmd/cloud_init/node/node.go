// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package node

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// nodeCmd represents the "cloud-init node" command
	var nodeCmd = &cobra.Command{
		Use:   "node",
		Args:  cobra.NoArgs,
		Short: "Manage cloud-init node-specific config",
		Long: `Manage cloud-init node-specific config.

See ochami-cloud-init(1) for more details.`,
		RunE: cli.PrintUsage,
	}

	// Add subcommands
	nodeCmd.AddCommand(
		newCmdNodeGet(),
		newCmdNodeSet(),
	)

	return nodeCmd
}
