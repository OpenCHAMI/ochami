// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package defaults

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
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
		RunE: cli.PrintUsage,
	}

	// Create flags
	metadataDefaultsCmd.PersistentFlags().BoolP("envelope", "e", false, "use the envelope (advanced) API, preserving metadata/labels/annotations, instead of the simple API")

	// Add subcommands
	metadataDefaultsCmd.AddCommand(
		newCmdMetadataDefaultsAdd(),
		newCmdMetadataDefaultsDelete(),
		newCmdMetadataDefaultsGet(),
		newCmdMetadataDefaultsList(),
		newCmdMetadataDefaultsPatch(),
		newCmdMetadataDefaultsSet(),
	)

	return metadataDefaultsCmd
}
