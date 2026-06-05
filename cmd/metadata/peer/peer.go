// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package peer

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// metadataPeerCmd represents the "metadata peer" command
	var metadataPeerCmd = &cobra.Command{
		Use:   "peer",
		Args:  cobra.NoArgs,
		Short: "Manage WireGuard peer configurations",
		Long: `Manage WireGuard peer configurations in the metadata service. This is a metacommand.
Commands under this one interact with the metadata-service
WireGuard peer endpoint.

See ochami-metadata(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
				os.Exit(0)
			}
		},
	}

	// Add subcommands
	metadataPeerCmd.AddCommand(
		newCmdMetadataPeerAdd(),
		newCmdMetadataPeerDelete(),
		newCmdMetadataPeerGet(),
		newCmdMetadataPeerList(),
		newCmdMetadataPeerPatch(),
		newCmdMetadataPeerSet(),
	)

	return metadataPeerCmd
}
