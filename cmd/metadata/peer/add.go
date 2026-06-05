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

func newCmdMetadataPeerAdd() *cobra.Command {
	// metadataPeerAddCmd represents the "metadata peer add" command
	var metadataPeerAddCmd = &cobra.Command{
		Use:   "add",
		Args:  cobra.NoArgs,
		Short: "Add one or more WireGuard peers to metadata-service",
		Long: `Add one or more WireGuard peers to metadata-service.

See ochami-metadata(1) for more details.`,
		Example: `  # Add WireGuard peer using JSON
  ochami metadata peer add -d \
    '{
       "public_key": "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=",
       "allowed_ip": "10.42.1.1/32",
       "description": "Peer for nid001000"
     }'

  # Add peer from YAML
  ochami metadata peer add -f yaml <<'EOF'
  public_key: xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=
  allowed_ip: 10.42.1.1/32
  description: Compute node peer
  EOF

  # Add multiple WireGuard peers using JSON array
  ochami metadata peer add -d \
    '[
       {
         "public_key": "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=",
         "allowed_ip": "10.42.1.1/32"
       },
       {
         "public_key": "yUJCB6sbcpVwoI5iupekc7f798RkMFSu2OBC5nArq9Eh=",
         "allowed_ip": "10.42.1.2/32"
       }
     ]'

  # Add multiple WireGuard peers using YAML array
  ochami metadata peer add -f yaml <<'EOF'
  - public_key: "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg="
    allowed_ip: "10.42.1.1/32"
  - public_key: "yUJCB6sbcpVwoI5iupekc7f798RkMFSu2OBC5nArq9Eh="
    allowed_ip: "10.42.1.2/32"
  EOF

  # Add multiple peers from file
  ochami metadata peer add -d @peers.json
  ochami metadata peer add -d @peers.yaml -f yaml

  # Add peers using data from stdin
  echo '<json_data>' | ochami metadata peer add -d @-
  echo '<json_data>' | ochami metadata peer add
  echo '<yaml_data>' | ochami metadata peer add -f yaml -d @-
  echo '<yaml_data>' | ochami metadata peer add -f yaml`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			metadataServiceClient := metadata_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Read peer data
			peers := []metadata_service_client.CreateWireGuardPeerRequest{}
			if cmd.Flag("data").Changed {
				cli.HandlePayloadSlice[metadata_service_client.CreateWireGuardPeerRequest](cmd, &peers)
			} else {
				cli.HandlePayloadStdinSlice[metadata_service_client.CreateWireGuardPeerRequest](cmd, &peers)
			}

			// Send off requests
			peersCreated, errs, err := metadataServiceClient.AddWireGuardPeers(cli.Token, peers)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to add WireGuard peers")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Deal with per-request errors
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to add WireGuard peer")
					errorsOccurred = true
				}
			}
			log.Logger.Debug().Msgf("WireGuard peers created: %+v", peersCreated)
			if errorsOccurred {
				cli.LogHelpError(cmd)
				log.Logger.Warn().Msg("WireGuard peer addition completed with errors")
				os.Exit(1)
			}
		},
	}

	// Create flags
	metadataPeerAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	metadataPeerAddCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	metadataPeerAddCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return metadataPeerAddCmd
}
