// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package console

import (
	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// consoleCmd represents the "rcs console" command
	var consoleCmd = &cobra.Command{
		Use:   "console",
		Short: "Console operations",
		Long: `Console operations for remote-console.

See ochami-rcs(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			cli.PrintUsageHandleError(cmd)
		},
	}

	// Add subcommands
	consoleCmd.AddCommand(
		newListCmd(),
		newShowCmd(),
		newConnectCmd(),
	)

	return consoleCmd
}
