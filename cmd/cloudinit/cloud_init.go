// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloudinit

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"

	// Subcommands
	defaultscmd "github.com/openchami/ochami/cmd/cloudinit/defaults"
	groupcmd "github.com/openchami/ochami/cmd/cloudinit/group"
	nodecmd "github.com/openchami/ochami/cmd/cloudinit/node"
	servicecmd "github.com/openchami/ochami/cmd/cloudinit/service"
)

func NewCmd() *cobra.Command {
	// cloudInitCmd represents the "cloud-init" command
	var cloudInitCmd = &cobra.Command{
		Use:   "cloud-init",
		Args:  cobra.NoArgs,
		Short: "Interact with the cloud-init service",
		Long: `Interact with the cloud-init service. This is a metacommand.

See ochami-cloud-init(1) for more details.`,
		RunE: cli.PrintUsage,
	}

	// Create flags
	cloudInitCmd.PersistentFlags().String("uri", "", "absolute base URI or relative base path of cloud-init")

	// Add subcommands
	cloudInitCmd.AddCommand(
		defaultscmd.NewCmd(),
		groupcmd.NewCmd(),
		nodecmd.NewCmd(),
		servicecmd.NewCmd(),
	)

	return cloudInitCmd
}
