// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package peer

import (
	"os"

	metadata_service_client "github.com/OpenCHAMI/metadata-service/pkg/client"
	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	metadata_service_lib "github.com/OpenCHAMI/ochami/internal/cli/metadata_service"
	"github.com/OpenCHAMI/ochami/internal/log"
)

func newCmdMetadataPeerSet() *cobra.Command {
	// metadataPeerSetCmd represents the "metadata peer set" command
	var metadataPeerSetCmd = &cobra.Command{
		Use:   "set <uid>",
		Args:  cobra.ExactArgs(1),
		Short: "Set details of an existing WireGuard peer spec",
		Long: `Set details of an existing WireGuard peer spec.

See ochami-metadata(1) for more details.`,
		Example: `  # Set WireGuard peer details using payload data
  ochami metadata peer set wireguardpeer-d614b918 -d \
    '{
       "metadata": {
         "name": "peer-nid001000"
       },
       "spec": {
         "public_key": "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=",
         "allowed_ip": "10.42.1.1/32",
         "description": "Updated peer"
       }
     }'

  # Set WireGuard peer details using YAML payload data
  ochami metadata peer set wireguardpeer-d614b918 -f yaml <<'EOF'
  metadata:
    name: peer-nid001000
  spec:
    public_key: "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg="
    allowed_ip: "10.42.1.1/32"
    description: "Updated peer"
  EOF

  # Set WireGuard peer details using file
  ochami metadata peer set wireguardpeer-d614b918 -d @peer.json
  ochami metadata peer set wireguardpeer-d614b918 -d @peer.yaml -f yaml

  # Set WireGuard peer details using data from stdin
  echo '<json_data>' | ochami metadata peer set wireguardpeer-d614b918 -d @-
  echo '<json_data>' | ochami metadata peer set wireguardpeer-d614b918
  echo '<yaml_data>' | ochami metadata peer set wireguardpeer-d614b918 -f yaml -d @-
  echo '<yaml_data>' | ochami metadata peer set wireguardpeer-d614b918 -f yaml`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			metadataServiceClient := metadata_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Read peer data
			peer := metadata_service_client.UpdateWireGuardPeerRequest{}
			cli.HandlePayload(cmd, &peer)

			// Send off requests
			peerSet, err := metadataServiceClient.SetWireGuardPeer(cli.Token, args[0], peer)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to set WireGuard peer")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Check that a modified item was returned
			if peerSet == nil {
				log.Logger.Error().Msg("WireGuard peer set returned no resource")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Print UIDs of modified items
			log.Logger.Info().Msgf("WireGuard peers set: %+v", []string{peerSet.Metadata.UID})
		},
	}

	// Create flags
	metadataPeerSetCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	metadataPeerSetCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	metadataPeerSetCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return metadataPeerSetCmd
}
