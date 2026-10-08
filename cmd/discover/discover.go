// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package discover

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"

	// Subcomands
	static_cmd "github.com/openchami/ochami/cmd/discover/static"
)

func NewCmd() *cobra.Command {
	// discoverCmd represents the discover command
	var discoverCmd = &cobra.Command{
		Use:   "discover",
		Args:  cobra.NoArgs,
		Short: "Perform static or dynamic discovery of nodes",
		RunE:  cli.PrintUsage,
	}

	// Add subcommands
	discoverCmd.AddCommand(
		static_cmd.NewCmd(),
	)

	return discoverCmd
}
