// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
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
		RunE: cli.PrintUsage,
	}

	// Create flags
	bootConfigCmd.PersistentFlags().BoolP("envelope", "e", false, "use the envelope (advanced) API, preserving metadata/labels/annotations, instead of the simple API")

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
