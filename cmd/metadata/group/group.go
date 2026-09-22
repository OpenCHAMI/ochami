// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// metadataGroupCmd represents the "metadata group" command
	var metadataGroupCmd = &cobra.Command{
		Use:   "group",
		Args:  cobra.NoArgs,
		Short: "Manage cloud-init group templates",
		Long: `Manage cloud-init group templates in the metadata service. This is a metacommand.
Commands under this one interact with the metadata-service
group endpoint.

See ochami-metadata(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cli.PrintUsageHandleError(cmd)
			}
			return nil
		},
	}

	// Create flags
	metadataGroupCmd.PersistentFlags().BoolP("envelope", "e", false, "use the envelope (advanced) API, preserving metadata/labels/annotations, instead of the simple API")

	// Add subcommands
	metadataGroupCmd.AddCommand(
		newCmdMetadataGroupAdd(),
		newCmdMetadataGroupDelete(),
		newCmdMetadataGroupGet(),
		newCmdMetadataGroupList(),
		newCmdMetadataGroupPatch(),
		newCmdMetadataGroupSet(),
	)

	return metadataGroupCmd
}
