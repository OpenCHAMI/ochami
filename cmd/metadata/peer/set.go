// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package peer

import (
	metadata_service_client "github.com/openchami/metadata-service/pkg/client"
	"github.com/spf13/cobra"

	api "github.com/openchami/metadata-service/apis/cloud-init.openchami.io/v1"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client/metadata_service"

	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
)

// metadataPeerSetOptions holds the flag values for the metadata peer set command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type metadataPeerSetOptions struct {
	Envelope bool
}

// runCoreMetadataPeerSet contains the core logic for the metadata peer set command.
// It takes the parsed options and performs the actual work of setting WireGuard peer details.
func runCoreMetadataPeerSet(cmd *cobra.Command, opts *metadataPeerSetOptions, args []string, metadataServiceClient *metadata_service.MetadataServiceClient, rt *cli.Runtime) error {
	// Handle token for this command
	if err := rt.HandleToken(cmd); err != nil {
		return err
	}

	// Determine how to read payload (simple versus advanced API)
	var peerSet *api.WireGuardPeer
	var reqErr error
	if opts.Envelope {
		// Use advanced API (spec, metadata, annotations)

		// Read peer data
		peer := metadata_service_client.UpdateWireGuardPeerRequest{}
		if cmd.Flag("data").Changed {
			if err := rt.HandlePayload(cmd, &peer); err != nil {
				return err
			}
		} else {
			if err := rt.HandlePayloadStdin(cmd, &peer); err != nil {
				return err
			}
		}

		// Send off request
		peerSet, reqErr = metadataServiceClient.SetWireGuardPeer(cmd.Context(), rt.Token, args[0], peer)
	} else {
		// Use simple API (spec)

		// Read peer data
		spec := api.WireGuardPeerSpec{}
		if cmd.Flag("data").Changed {
			if err := rt.HandlePayload(cmd, &spec); err != nil {
				return err
			}
		} else {
			if err := rt.HandlePayloadStdin(cmd, &spec); err != nil {
				return err
			}
		}

		// Send off request
		peerSet, reqErr = metadataServiceClient.SetWireGuardPeerSpec(cmd.Context(), rt.Token, args[0], spec)
	}
	if reqErr != nil {
		return cli.ClassifyClientError(reqErr, "failed to set WireGuard peer", "failed to set WireGuard peer")

	}

	// Check that a modified item was returned
	if peerSet == nil {
		return cli.Errorf(cli.CodeGeneric, "WireGuard peer set returned no resource")
	}

	log.Logger.Debug().Msgf("WireGuard peer set: %+v", peerSet)

	return nil
}

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
        "public_key": "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=",
        "allowed_ip": "10.42.1.1/32",
        "description": "Updated peer"
      }'

  # Set WireGuard peer details using YAML payload data
  ochami metadata peer set wireguardpeer-d614b918 -f yaml <<'EOF'
    public_key: "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg="
    allowed_ip: "10.42.1.1/32"
    description: "Updated peer"
    EOF'

  # Set WireGuard peer details preserving labels/annotations (envelope API)
  ochami metadata peer set wireguardpeer-d614b918 -e -d \
     '{
        "metadata": {
          "labels": {
            "env": "prod"
          }
        },
        "spec": {
          "public_key": "xTIBA5rboUvnH4htodjb6e6e97QjLERt1NAB4mZqp8Dg=",
          "allowed_ip": "10.42.1.1/32"
        }
      }'

  # Set WireGuard peer details using file
  ochami metadata peer set wireguardpeer-d614b918 -d @peer.json
  odami metadata peer set wireguardpeer-d614b918 -d @peer.yaml -f yaml

  # Set WireGuard peer details using data from stdin
  echo '<json_data>' | odami metadata peer set wireguardpeer-d614b918 -d @-
  echo '<json_data>' | odami metadata peer set wireguardpeer-d614b918
  echo '<yaml_data>' | odami metadata peer set wireguardpeer-d614b918 -f yaml -d @-
  echo '<yaml_data>' | odami metadata peer set wireguardpeer-d614b918 -f yaml`,
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
			opts := &metadataPeerSetOptions{}
			if cmd.Flag("envelope").Changed {
				opts.Envelope, _ = cmd.Flags().GetBool("envelope") //nolint:errcheck // Flag registered with matching type, error impossible
			}

			return runCoreMetadataPeerSet(cmd, opts, args, metadataServiceClient, rt)
		},
	}

	// Create flags
	metadataPeerSetCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")

	cli.AddFormatInputFlag(metadataPeerSetCmd)
	metadataPeerSetCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return metadataPeerSetCmd
}
