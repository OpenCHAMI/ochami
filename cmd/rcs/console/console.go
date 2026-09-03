// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package console

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// consoleCmd represents the "rcs console" command
	var consoleCmd = &cobra.Command{
		Use:   "console",
		Short: "Console operations",
		Long: `Console operations for remote-console.

See ochami-rcs(1) for more details.`,
		RunE: cli.PrintUsage,
	}

	// Add subcommands
	consoleCmd.AddCommand(
		newConnectCmd(),
		newListCmd(),
		newShowCmd(),
	)

	return consoleCmd
}
