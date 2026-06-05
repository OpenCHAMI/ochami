// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	metadata_service_lib "github.com/OpenCHAMI/ochami/internal/cli/metadata_service"
	"github.com/OpenCHAMI/ochami/internal/log"
)

func newCmdMetadataGroupList() *cobra.Command {
	// metadataGroupListCmd represents the "metadata group list" command
	var metadataGroupListCmd = &cobra.Command{
		Use:   "list",
		Args:  cobra.NoArgs,
		Short: "List groups",
		Long: `List groups.

See ochami-metadata(1) for more details.`,
		Example: `  # List all groups
  ochami metadata group list

  # List groups in YAML format
  ochami metadata group list -F yaml`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			metadataServiceClient := metadata_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Make request
			outBytes, err := metadataServiceClient.ListGroups(cli.Token, cli.FormatOutput)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to list groups")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Print output
			fmt.Print(string(outBytes))
		},
	}

	// Create flags
	metadataGroupListCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	metadataGroupListCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return metadataGroupListCmd
}
