// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package defaults

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// metadataDefaultsCmd represents the "metadata defaults" command
	var metadataDefaultsCmd = &cobra.Command{
		Use:   "defaults",
		Args:  cobra.NoArgs,
		Short: "Manage default metadata values for a cluster",
		Long: `Manage default metadata values for a cluster. This is a metacommand.
Commands under this one interact with the metadata-service
cluster defaults endpoint.

See ochami-metadata(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
				os.Exit(0)
			}
		},
	}

	// Add subcommands
	metadataDefaultsCmd.AddCommand(
		newCmdMetadataDefaultsAdd(),
		newCmdMetadataDefaultsList(),
		newCmdMetadataDefaultsSet(),
	)

	return metadataDefaultsCmd
}
