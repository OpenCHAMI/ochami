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

func newCmdMetadataInstanceList() *cobra.Command {
	// metadataInstanceListCmd represents the "metadata instance list" command
	var metadataInstanceListCmd = &cobra.Command{
		Use:   "list",
		Args:  cobra.NoArgs,
		Short: "List instance infos",
		Long: `List instance infos.

See ochami-metadata(1) for more details.`,
		Example: `  # List all instance infos
  ochami metadata instance list

  # List instance infos in YAML format
  ochami metadata instance list -F yaml`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			metadataServiceClient := metadata_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Make request
			outBytes, err := metadataServiceClient.ListInstanceInfos(cli.Token, cli.FormatOutput)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to list instance infos")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Print output
			fmt.Print(string(outBytes))
		},
	}

	// Create flags
	metadataInstanceListCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	metadataInstanceListCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return metadataInstanceListCmd
}
