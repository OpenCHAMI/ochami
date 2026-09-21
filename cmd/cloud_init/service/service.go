// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package service

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// serviceCmd represents the "cloud-init service" command
	var serviceCmd = &cobra.Command{
		Use:   "service",
		Args:  cobra.NoArgs,
		Short: "Manage and check cloud-init itself",
		Long: `Manage and check cloud-init itself. This is a metacommand.

See ochami-cloud-init(1) for more details.`,
		RunE: cli.PrintUsage,
	}

	// Add subcommands
	serviceCmd.AddCommand(
		newCmdServiceStatus(),
		newCmdServiceVersion(),
	)

	return serviceCmd
}
