// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package instance

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	metadata_service_lib "github.com/OpenCHAMI/ochami/internal/cli/metadata_service"
	"github.com/OpenCHAMI/ochami/internal/log"
)

func newCmdMetadataInstanceGet() *cobra.Command {
	// metadataInstanceGetCmd represents the "metadata instance get" command
	var metadataInstanceGetCmd = &cobra.Command{
		Use:   "get <uid>",
		Args:  cobra.ExactArgs(1),
		Short: "Get an instance info by its UID",
		Long: `Get an instance info by its UID.

See ochami-metadata(1) for more details.`,
		Example: `  # Get info about an instance
  ochami metadata instance get instanceinfo-773d99bf

  # Get instance info in YAML format
  ochami metadata instance get instanceinfo-773d99bf -F yaml`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			metadataServiceClient := metadata_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			uid := args[0]

			// Make request
			outBytes, err := metadataServiceClient.GetInstanceInfo(cli.Token, cli.FormatOutput, uid)
			if err != nil {
				log.Logger.Error().Err(err).Msgf("failed to get instance info for %s", uid)
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Print output
			fmt.Print(string(outBytes))
		},
	}

	// Create flags
	metadataInstanceGetCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	metadataInstanceGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return metadataInstanceGetCmd
}
