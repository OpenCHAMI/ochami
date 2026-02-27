// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// bootConfigCmd represents the "boot config" command
	var bootConfigCmd = &cobra.Command{
		Use:   "config",
		Args:  cobra.NoArgs,
		Short: "Manage node and BMC boot configuration",
		Long: `Manage node and BMC boot configuration, including kernel/initrd
URI and kernel command line arguments. This is a metacommand.

See ochami-boot(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
				os.Exit(0)
			}
		},
	}

	// Add subcommands
	bootConfigCmd.AddCommand(
		newCmdBootConfigAdd(),
		newCmdBootConfigDelete(),
		newCmdBootConfigGet(),
		newCmdBootConfigList(),
		newCmdBootConfigPatch(),
		newCmdBootConfigSet(),
	)

	return bootConfigCmd
}
