// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package boot

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"

	// Subcommands
	image_cmd "github.com/OpenCHAMI/ochami/cmd/bss/boot/image"
	params_cmd "github.com/OpenCHAMI/ochami/cmd/bss/boot/params"
	script_cmd "github.com/OpenCHAMI/ochami/cmd/bss/boot/script"
)

func NewCmd() *cobra.Command {
	// bootCmd represents the "bss boot" command
	var bootCmd = &cobra.Command{
		Use:   "boot",
		Args:  cobra.NoArgs,
		Short: "Manage boot configuration for components",
		Long: `Manage boot configuration for components. This is a metacommand. Commands
under this one interact with the Boot Script Service (BSS).

See ochami-bss(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
				os.Exit(0)
			}
		},
	}

	// Add subcommands
	bootCmd.AddCommand(
		image_cmd.NewCmd(),
		params_cmd.NewCmd(),
		script_cmd.NewCmd(),
	)

	return bootCmd
}
