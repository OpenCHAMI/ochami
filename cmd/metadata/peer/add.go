// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package peer

import (
	metadata_service_client "github.com/openchami/metadata-service/pkg/client"
	"github.com/spf13/cobra"

	api "github.com/openchami/metadata-service/apis/cloud-init.openchami.io/v1"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/metadata_service"

	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
)

// metadataPeerAddOptions holds the flag values for the metadata peer add command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type metadataPeerAddOptions struct {
	Envelope bool
}

// runCoreMetadataPeerAdd contains the core logic for the metadata peer add command.
// It takes the parsed options and performs the actual work of adding WireGuard peers.
func runCoreMetadataPeerAdd(cmd *cobra.Command, opts *metadataPeerAddOptions, metadataServiceClient *metadata_service.MetadataServiceClient, rt *cli.Runtime) error {
	// Handle token for this command
	if err := rt.HandleToken(cmd); err != nil {
		return err
	}

	// Determine how to read payload (simple versus advanced API)
	var results client.BatchResult[api.WireGuardPeer]
	if opts.Envelope {
		// Use advanced API (spec, metadata, annotations)

		// Read peer data
		peers := []metadata_service_client.CreateWireGuardPeerRequest{}
		if cmd.Flag("data").Changed {
			if err := cli.HandlePayloadSliceWithRuntime[metadata_service_client.CreateWireGuardPeerRequest](rt, cmd, &peers); err != nil {
				return err
			}
		} else {
			if err := cli.HandlePayloadStdinSliceWithRuntime[metadata_service_client.CreateWireGuardPeerRequest](rt, cmd, &peers); err != nil {
				return err
			}
		}

		// Send off requests
		results = metadataServiceClient.AddWireGuardPeers(cmd.Context(), rt.Token, peers)
	} else {
		// Use simple API (spec)

		// Read peer data
		peers := []metadata_service.WireGuardPeerSpec{}
		if cmd.Flag("data").Changed {
			if err := cli.HandlePayloadSliceWithRuntime[metadata_service.WireGuardPeerSpec](rt, cmd, &peers); err != nil {
				return err
			}
		} else {
			if err := cli.HandlePayloadStdinSliceWithRuntime[metadata_service.WireGuardPeerSpec](rt, cmd, &peers); err != nil {
				return err
			}
		}

		// Send off requests
		results = metadataServiceClient.AddWireGuardPeerSpecs(cmd.Context(), rt.Token, peers)
	}

	// Deal with per-request errors
	var reqErrorsOccurred = false
	for _, err := range results.Errors() {
		if err != nil {
			rt.Logger.Error().Err(err).Msg("failed to add WireGuard peer")
			reqErrorsOccurred = true
		}
	}
	var names []string
	for _, peer := range results.Values() {
		names = append(names, peer.Metadata.Name)
	}
	rt.Logger.Debug().Msgf("WireGuard peers created: %q", names)
	if reqErrorsOccurred {
		return cli.Errorf(cli.CodeHTTP, "WireGuard peer addition completed with errors")
	}

	return nil
}

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
   EOF

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
   EOF

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

			// Extract options from flags
			// Since flags are registered with the correct types on this command,
			// these Get* calls cannot fail, so we ignore errors with explicit comments
			opts := &metadataPeerAddOptions{}
			if cmd.Flag("envelope").Changed {
				opts.Envelope, _ = cmd.Flags().GetBool("envelope") //nolint:errcheck // Flag registered with matching type, error impossible
			}

			return runCoreMetadataPeerAdd(cmd, opts, metadataServiceClient, rt)
		},
	}

	// Create flags
	metadataPeerAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")

	cli.AddFormatInputFlag(metadataPeerAddCmd)
	metadataPeerAddCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return metadataPeerAddCmd
}
