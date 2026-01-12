// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package script

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// bootScriptCmd represents the "bss boot script" command
	var bootScriptCmd = &cobra.Command{
		Use:   "script",
		Args:  cobra.NoArgs,
		Short: "Work with boot scripts for components",
		Long: `Work with boot scripts for components. This is a metacommand.

See ochami-bss(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
				os.Exit(0)
			}
		},
	}

	// Add subcommands
	bootScriptCmd.AddCommand(
		newCmdBootScriptGet(),
	)

	return bootScriptCmd
}
