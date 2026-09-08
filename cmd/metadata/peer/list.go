// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package peer

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
)

func newCmdMetadataPeerList() *cobra.Command {
	// metadataPeerListCmd represents the "metadata peer list" command
	var metadataPeerListCmd = &cobra.Command{
		Use:   "list",
		Args:  cobra.NoArgs,
		Short: "List WireGuard peers",
		Long: `List WireGuard peers.

See ochami-metadata(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			metadataServiceClient, err := metadata_service_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// Make request
			outBytes, err := metadataServiceClient.ListWireGuardPeers(cli.Token, cli.FormatOutput)
			if err != nil {
				return cli.ClassifyClientError(err, "failed to list WireGuard peers", "failed to list WireGuard peers")

			}

			// Print output
			fmt.Fprint(cli.Ios.Out(), string(outBytes))

			return nil
		},
	}

	// Create flags
	metadataPeerListCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	metadataPeerListCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return metadataPeerListCmd
}
