// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package peer

import (
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
		Example: `  # List all WireGuard peers
  ochami metadata peer list

  # List WireGuard peers in YAML format
  ochami metadata peer list -F yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Create client to use for requests
			metadataServiceClient, err := metadata_service_lib.GetClientWithRuntime(cmd, rt)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := rt.HandleToken(cmd); err != nil {
				return err
			}

			// Make request
			outBytes, err := metadataServiceClient.ListWireGuardPeers(cmd.Context(), rt.Token, rt.FormatOutput)
			if err != nil {
				return cli.ClassifyClientError(err, "failed to list WireGuard peers", "failed to list WireGuard peers")

			}

			// Print output
			if err := cli.WriteOutput(rt.Ios.Out(), outBytes); err != nil {
				return err
			}

			return nil
		},
	}

	// Create flags

	cli.AddFormatOutputFlag(metadataPeerListCmd)
	metadataPeerListCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return metadataPeerListCmd
}
