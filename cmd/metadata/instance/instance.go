// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package instance

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// metadataInstanceCmd represents the "metadata instance" command
	var metadataInstanceCmd = &cobra.Command{
		Use:   "instance",
		Args:  cobra.NoArgs,
		Short: "Manage instance information",
		Long: `Manage instance information in the metadata service. This is a metacommand.
Commands under this one interact with the metadata-service
instance info endpoint.

See ochami-metadata(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
				os.Exit(0)
			}
		},
	}

	// Add subcommands
	metadataInstanceCmd.AddCommand(
		newCmdMetadataInstanceAdd(),
		newCmdMetadataInstanceDelete(),
		newCmdMetadataInstanceGet(),
		newCmdMetadataInstanceList(),
		newCmdMetadataInstancePatch(),
		newCmdMetadataInstanceSet(),
	)

	return metadataInstanceCmd
}
