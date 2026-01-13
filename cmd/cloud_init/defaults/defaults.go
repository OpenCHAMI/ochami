// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package defaults

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// defaultsCmd represents the "cloud-init defaults" command
	var defaultsCmd = &cobra.Command{
		Use:   "defaults",
		Args:  cobra.NoArgs,
		Short: "View cloud-init default values for the cluster",
		Long: `View default meta-data values for a cluster. This is a metacommand.
Commands under this one interact with the cloud-init
cluster-defaults endpoint.

See ochami-cloud-init(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
				os.Exit(0)
			}
		},
	}

	// Add subcommands
	defaultsCmd.AddCommand(
		newCmdDefaultsGet(),
		newCmdDefaultsSet(),
	)

	return defaultsCmd
}
