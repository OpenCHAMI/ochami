// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package peer

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	metadata_service_lib "github.com/OpenCHAMI/ochami/internal/cli/metadata_service"
	"github.com/OpenCHAMI/ochami/internal/log"
)

func newCmdMetadataPeerList() *cobra.Command {
	// metadataPeerListCmd represents the "metadata peer list" command
	var metadataPeerListCmd = &cobra.Command{
		Use:   "list",
		Args:  cobra.NoArgs,
		Short: "List WireGuard peers",
		Long: `List WireGuard peers.

See ochami-metadata(1) for more details.`,
		Example: `  # List all WireGuard peers
  ochami metadata peer list

  # List WireGuard peers in YAML format
  ochami metadata peer list -F yaml`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			metadataServiceClient := metadata_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Make request
			outBytes, err := metadataServiceClient.ListWireGuardPeers(cli.Token, cli.FormatOutput)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to list WireGuard peers")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Print output
			fmt.Print(string(outBytes))
		},
	}

	// Create flags
	metadataPeerListCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	metadataPeerListCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return metadataPeerListCmd
}
