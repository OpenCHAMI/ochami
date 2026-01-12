// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"

	// Subcommands
	defaults_cmd "github.com/OpenCHAMI/ochami/cmd/cloud_init/defaults"
	group_cmd "github.com/OpenCHAMI/ochami/cmd/cloud_init/group"
	node_cmd "github.com/OpenCHAMI/ochami/cmd/cloud_init/node"
	service_cmd "github.com/OpenCHAMI/ochami/cmd/cloud_init/service"
)

func NewCmd() *cobra.Command {
	// cloudInitCmd represents the "cloud-init" command
	var cloudInitCmd = &cobra.Command{
		Use:   "cloud-init",
		Args:  cobra.NoArgs,
		Short: "Interact with the cloud-init service",
		Long: `Interact with the cloud-init service. This is a metacommand.

See ochami-cloud-init(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
				os.Exit(0)
			}
		},
	}

	// Create flags
	cloudInitCmd.PersistentFlags().String("uri", "", "absolute base URI or relative base path of cloud-init")

	// Add subcommands
	cloudInitCmd.AddCommand(
		defaults_cmd.NewCmd(),
		group_cmd.NewCmd(),
		node_cmd.NewCmd(),
		service_cmd.NewCmd(),
	)

	return cloudInitCmd
}
