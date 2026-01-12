// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package discover

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"

	// Subcomands
	static_cmd "github.com/OpenCHAMI/ochami/cmd/discover/static"
)

func NewCmd() *cobra.Command {
	// discoverCmd represents the discover command
	var discoverCmd = &cobra.Command{
		Use:   "discover",
		Args:  cobra.NoArgs,
		Short: "Perform static or dynamic discovery of nodes",
		Run: func(cmd *cobra.Command, args []string) {
			// Check that all required args are passed
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
				os.Exit(0)
			}
		},
	}

	// Add subcommands
	discoverCmd.AddCommand(
		static_cmd.NewCmd(),
	)

	return discoverCmd
}
