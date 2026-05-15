// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package service

import (
	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	var serviceCmd = &cobra.Command{
		Use:   "service",
		Short: "Console service operations",
		Long: `Console service operations for remote-console.

See ochami-rcs(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			cli.PrintUsageHandleError(cmd)
		},
	}

	serviceCmd.AddCommand(
		newStatusCmd(),
	)

	return serviceCmd
}
