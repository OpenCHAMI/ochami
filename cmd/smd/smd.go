// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package smd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"

	// Subcommands
	compep_cmd "github.com/OpenCHAMI/ochami/cmd/smd/compep"
	component_cmd "github.com/OpenCHAMI/ochami/cmd/smd/component"
	group_cmd "github.com/OpenCHAMI/ochami/cmd/smd/group"
	iface_cmd "github.com/OpenCHAMI/ochami/cmd/smd/iface"
	rfe_cmd "github.com/OpenCHAMI/ochami/cmd/smd/rfe"
	service_cmd "github.com/OpenCHAMI/ochami/cmd/smd/service"
	status_cmd "github.com/OpenCHAMI/ochami/cmd/smd/status" // DEPRECATED
)

func NewCmd() *cobra.Command {
	// smdCmd represents the bss command
	var smdCmd = &cobra.Command{
		Use:   "smd",
		Args:  cobra.NoArgs,
		Short: "Communicate with the State Management Database (SMD)",
		Long: `Communicate with the State Management Database (SMD).

See ochami-smd(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
				os.Exit(0)
			}
		},
	}

	// Create flags
	smdCmd.PersistentFlags().String("uri", "", "absolute base URI or relative base path of SMD")

	// Add subcommands
	smdCmd.AddCommand(
		compep_cmd.NewCmd(),
		component_cmd.NewCmd(),
		group_cmd.NewCmd(),
		iface_cmd.NewCmd(),
		rfe_cmd.NewCmd(),
		service_cmd.NewCmd(),
		status_cmd.NewCmd(), // DEPRECATED
	)

	return smdCmd
}
