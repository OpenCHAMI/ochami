// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"

	// Subcommands
	defaults_cmd "github.com/openchami/ochami/cmd/cloud_init/defaults"
	group_cmd "github.com/openchami/ochami/cmd/cloud_init/group"
	node_cmd "github.com/openchami/ochami/cmd/cloud_init/node"
	render_cmd "github.com/openchami/ochami/cmd/cloud_init/render"
	service_cmd "github.com/openchami/ochami/cmd/cloud_init/service"
)

func NewCmd() *cobra.Command {
	// cloudInitCmd represents the "cloud-init" command
	var cloudInitCmd = &cobra.Command{
		Use:   "cloud-init",
		Args:  cobra.NoArgs,
		Short: "Manage cloud-init data and render templates",
		Long: `Manage cloud-init data and render templates. This is a metacommand.

See ochami-cloud-init(1) for more details.`,
		RunE: cli.PrintUsage,
	}

	// Create flags
	cloudInitCmd.PersistentFlags().String("uri", "", "absolute base URI or relative base path of cloud-init")

	// Add subcommands
	cloudInitCmd.AddCommand(
		defaults_cmd.NewCmd(),
		group_cmd.NewCmd(),
		node_cmd.NewCmd(),
		render_cmd.NewCmd(),
		service_cmd.NewCmd(),
	)

	return cloudInitCmd
}
