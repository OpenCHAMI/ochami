// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package peer

import (
	"errors"

	metadata_service_client "github.com/openchami/metadata-service/pkg/client"
	"github.com/spf13/cobra"

	api "github.com/openchami/metadata-service/apis/cloud-init.openchami.io/v1"

	"github.com/openchami/ochami/internal/cli"
	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/metadata_service"
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
        "name": "peer-nid001000",
        "public_key": "xTIBA5rboUvnH4htodjb6e6e97QjLERt1NAB4mZqp8Dg=",
        "allowed_ip": "10.42.1.1/32",
        "description": "Peer for nid001000"
      }'

  # Add peer from YAML
  ochami metadata peer add -f yaml <<'EOF'
    name: peer-nid001000
    public_key: xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=
    allowed_ip: 10.42.1.1/32
    description: Compute node peer
    EOF'

  # Add multiple WireGuard peers using JSON array of specs
  ochami metadata peer add -d \
     '[
        {
          "name": "peer-nid001000",
          "public_key": "xTIBA5rboUvnH4htodjb6e6e97QjLERt1NAB4mZqp8Dg=",
          "allowed_ip": "10.42.1.1/32"
        },
        {
          "name": "peer-nid001001",
          "public_key": "yUJCB6sbcpVwoI5iupekc7f798RkMFSu2OBC5nArq9Eh=",
          "allowed_ip": "10.42.1.2/32"
        }
      ]'

  # Add multiple WireGuard peers using YAML array of specs
  ochami metadata peer add -f yaml <<'EOF'
   - name: peer-nid001000
     public_key: "xTIBA5rboUvnH4htodjb6e6e97QjLERt1NAB4mZqp8Dg="
     allowed_ip: "10.42.1.1/32"
   - name: peer-nid001001
     public_key: "yUJCB6sbcpVwoI5iupekc7f798RkMFSu2OBC5nArq9Eh="
     allowed_ip: "10.42.1.2/32"
   EOF'

  # Add WireGuard peer preserving labels/annotations (envelope API)
  ochami metadata peer add -e -d \
     '{
        "metadata": {
          "name": "peer-nid001000",
          "labels": {
            "env": "prod"
          }
        },
        "spec": {
          "public_key": "xTIBA5rboUvnH4htodjb6e6e97QjLERt1NAB4mZqp8Dg=",
          "allowed_ip": "10.42.1.1/32"
        }
      }'

  # Add multiple peers from file
  ochami metadata peer add -d @peers.json
  ochami metadata peer add -d @peers.yaml -f yaml

  # Add peers using data from stdin
  echo '<json_data>' | ochami metadata peer add -d @-
  echo '<json_data>' | ochami metadata peer add
  echo '<yaml_data>' | ochami metadata peer add -f yaml -d @-
  echo '<yaml_data>' | ochami metadata peer add -f yaml`,
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

			// Determine how to read payload (simple versus advanced API)
			envelope, _ := cmd.Flags().GetBool("envelope")

			var peersCreated []api.WireGuardPeer
			var reqErrs []error
			var reqErr error
			if envelope {
				// Use advanced API (spec, metadata, annotations)

				// Read peer data
				peers := []metadata_service_client.CreateWireGuardPeerRequest{}
				if cmd.Flag("data").Changed {
					if err := cli.HandlePayloadSlice[metadata_service_client.CreateWireGuardPeerRequest](cmd, &peers); err != nil {
						return err
					}
				} else {
					if err := cli.HandlePayloadStdinSlice[metadata_service_client.CreateWireGuardPeerRequest](cmd, &peers); err != nil {
						return err
					}
				}

				// Send off requests
				peersCreated, reqErrs, reqErr = metadataServiceClient.AddWireGuardPeers(cli.Token, peers)
			} else {
				// Use simple API (spec)

				// Read peer data
				peers := []metadata_service.WireGuardPeerSpec{}
				if cmd.Flag("data").Changed {
					if err := cli.HandlePayloadSlice[metadata_service.WireGuardPeerSpec](cmd, &peers); err != nil {
						return err
					}
				} else {
					if err := cli.HandlePayloadStdinSlice[metadata_service.WireGuardPeerSpec](cmd, &peers); err != nil {
						return err
					}
				}

				// Send off requests
				peersCreated, reqErrs, reqErr = metadataServiceClient.AddWireGuardPeerSpecs(cli.Token, peers)
			}

			// Handle any non-request error
			if reqErr != nil {
				if errors.Is(reqErr, client.UnsuccessfulHTTPError) {
					return cli.Errorf(cli.CodeHTTP, "failed to add WireGuard peers: %w", reqErr)
				}
				return cli.Errorf(cli.CodeNetwork, "failed to add WireGuard peers: %w", reqErr)
			}

			// Deal with per-request errors
			var reqErrorsOccurred = false
			for _, err := range reqErrs {
				if err != nil {
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(err).Msg("failed to add WireGuard peer")
					} else {
						log.Logger.Error().Err(err).Msg("failed to add WireGuard peer")
					}
					reqErrorsOccurred = true
				}
			}

			// Print names of created items
			var names []string
			for _, peer := range peersCreated {
				names = append(names, peer.Metadata.Name)
			}
			log.Logger.Info().Msgf("WireGuard peers created: %q", names)

			// Warn if any request errors occurred
			if reqErrorsOccurred {
				return cli.Errorf(cli.CodeHTTP, "WireGuard peer addition completed with errors")
			}

			return nil
		},
	}

	// Create flags
	metadataPeerAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	metadataPeerAddCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	metadataPeerAddCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return metadataPeerAddCmd
}
