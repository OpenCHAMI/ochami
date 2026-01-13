// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package bss

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"

	// Subcommands
	boot_cmd "github.com/OpenCHAMI/ochami/cmd/bss/boot"
	dumpstate_cmd "github.com/OpenCHAMI/ochami/cmd/bss/dumpstate"
	history_cmd "github.com/OpenCHAMI/ochami/cmd/bss/history"
	hosts_cmd "github.com/OpenCHAMI/ochami/cmd/bss/hosts"
	service_cmd "github.com/OpenCHAMI/ochami/cmd/bss/service"
	status_cmd "github.com/OpenCHAMI/ochami/cmd/bss/status" // DEPRECATED
)

func NewCmd() *cobra.Command {
	// bssCmd represents the bss command
	var bssCmd = &cobra.Command{
		Use:   "bss",
		Args:  cobra.NoArgs,
		Short: "Communicate with the Boot Script Service (BSS)",
		Long: `Communicate with the Boot Script Service (BSS).

See ochami-bss(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
				os.Exit(0)
			}
		},
	}

	// Create flags
	bssCmd.PersistentFlags().String("uri", "", "absolute base URI or relative base path of BSS")

	// Add subcommands
	bssCmd.AddCommand(
		boot_cmd.NewCmd(),
		dumpstate_cmd.NewCmd(),
		history_cmd.NewCmd(),
		hosts_cmd.NewCmd(),
		service_cmd.NewCmd(),
		status_cmd.NewCmd(), // DEPRECATED
	)

	return bssCmd
}
