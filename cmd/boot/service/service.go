// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package service

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// bootServiceCmd represents the "boot service" command
	var bootServiceCmd = &cobra.Command{
		Use:   "service",
		Args:  cobra.NoArgs,
		Short: "Manage and check boot-service itself",
		Long: `Manage and check boot-service itself.

See ochami-boot(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cli.PrintUsageHandleError(cmd)
			}
			return nil
		},
	}

	// Add subcommands
	bootServiceCmd.AddCommand(
		newCmdServiceStatus(),
	)

	return bootServiceCmd
}
