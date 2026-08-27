// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package params

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// bootParamsCmd represents the "bss boot params" command
	var bootParamsCmd = &cobra.Command{
		Use:   "params",
		Args:  cobra.NoArgs,
		Short: "Work with boot parameters for components",
		Long: `Work with boot parameters for components, including kernel URI, initrd URI,
and kernel command line arguments. This is a metacommand.

See ochami-bss(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cli.PrintUsageHandleError(cmd)
			}
			return nil
		},
	}

	// Add subcommands
	bootParamsCmd.AddCommand(
		newCmdBootParamsAdd(),
		newCmdBootParamsDelete(),
		newCmdBootParamsGet(),
		newCmdBootParamsSet(),
		newCmdBootParamsUpdate(),
	)

	return bootParamsCmd
}
