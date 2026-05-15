// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rcs

import (
	"os"

	"github.com/spf13/cobra"

	console_cmd "github.com/OpenCHAMI/ochami/cmd/rcs/console"
	service_cmd "github.com/OpenCHAMI/ochami/cmd/rcs/service"
	"github.com/OpenCHAMI/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// rcsCmd represents the rcs command
	var rcsCmd = &cobra.Command{
		Use:   "rcs",
		Args:  cobra.NoArgs,
		Short: "Manage remote consoles",
		Long: `Manage remote consoles via the remote-console service.

See ochami-rcs(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			cli.PrintUsageHandleError(cmd)
			os.Exit(0)
		},
	}

	// Create flags
	rcsCmd.PersistentFlags().String("uri", "", "absolute base URI or relative base path of remote-console")

	// Add subcommands
	rcsCmd.AddCommand(
		console_cmd.NewCmd(),
		service_cmd.NewCmd(),
	)

	return rcsCmd
}
