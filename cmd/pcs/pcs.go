// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package pcs

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"

	// Subcommands
	service_cmd "github.com/OpenCHAMI/ochami/cmd/pcs/service"
	transition_cmd "github.com/OpenCHAMI/ochami/cmd/pcs/transition"
)

func NewCmd() *cobra.Command {
	// pcsCmd represents the pcs command
	var pcsCmd = &cobra.Command{
		Use:   "pcs",
		Args:  cobra.NoArgs,
		Short: "Interact with the Power Control Service (PCS)",
		Long: `Interact with the Power Control Service (PCS).

See ochami-pcs(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
				os.Exit(0)
			}
		},
	}

	// Create flags
	pcsCmd.PersistentFlags().String("uri", "", "absolute base URI or relative base path of PCS")

	// Add subcommands
	pcsCmd.AddCommand(
		service_cmd.NewCmd(),
		transition_cmd.NewCmd(),
	)

	return pcsCmd
}
