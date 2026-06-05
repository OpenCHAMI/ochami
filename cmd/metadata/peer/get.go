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

func newCmdMetadataPeerGet() *cobra.Command {
	// metadataPeerGetCmd represents the "metadata peer get" command
	var metadataPeerGetCmd = &cobra.Command{
		Use:   "get <uid>",
		Args:  cobra.ExactArgs(1),
		Short: "Get a WireGuard peer by its UID",
		Long: `Get a WireGuard peer by its UID.

See ochami-metadata(1) for more details.`,
		Example: `  # Get info about a WireGuard peer
  ochami metadata peer get wireguardpeer-773d99bf

  # Get WireGuard peer in YAML format
  ochami metadata peer get wireguardpeer-773d99bf -F yaml`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			metadataServiceClient := metadata_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			uid := args[0]

			// Make request
			outBytes, err := metadataServiceClient.GetWireGuardPeer(cli.Token, cli.FormatOutput, uid)
			if err != nil {
				log.Logger.Error().Err(err).Msgf("failed to get WireGuard peer info for %s", uid)
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Print output
			fmt.Print(string(outBytes))
		},
	}

	// Create flags
	metadataPeerGetCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	metadataPeerGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return metadataPeerGetCmd
}
