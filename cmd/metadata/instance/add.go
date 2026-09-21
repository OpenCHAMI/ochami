// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package instance

import (
	metadata_service_client "github.com/openchami/metadata-service/pkg/client"
	"github.com/spf13/cobra"

	api "github.com/openchami/metadata-service/apis/cloud-init.openchami.io/v1"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/metadata_service"

	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
)

// metadataInstanceAddOptions holds the flag values for the metadata instance add command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type metadataInstanceAddOptions struct {
	Envelope bool
}

// runCoreMetadataInstanceAdd contains the core logic for the metadata instance add command.
// It takes the parsed options and performs the actual work of adding instances.
func runCoreMetadataInstanceAdd(cmd *cobra.Command, opts *metadataInstanceAddOptions, metadataServiceClient *metadata_service.MetadataServiceClient, rt *cli.Runtime) error {
	// Handle token for this command
	if err := rt.HandleToken(cmd); err != nil {
		return err
	}

	// Determine how to read payload (simple versus advanced API)
	var results client.BatchResult[api.InstanceInfo]
	if opts.Envelope {
		// Use advanced API (spec, metadata, annotations)

		// Read instance data
		instances := []metadata_service_client.CreateInstanceInfoRequest{}
		if cmd.Flag("data").Changed {
			if err := cli.HandlePayloadSliceWithRuntime[metadata_service_client.CreateInstanceInfoRequest](rt, cmd, &instances); err != nil {
				return err
			}
		} else {
			if err := cli.HandlePayloadStdinSliceWithRuntime[metadata_service_client.CreateInstanceInfoRequest](rt, cmd, &instances); err != nil {
				return err
			}
		}

		// Send off requests
		results = metadataServiceClient.AddInstanceInfos(cmd.Context(), rt.Token, instances)
	} else {
		// Use simple API (spec)

		// Read instance data
		instances := []metadata_service.InstanceInfoSpec{}
		if cmd.Flag("data").Changed {
			if err := cli.HandlePayloadSliceWithRuntime[metadata_service.InstanceInfoSpec](rt, cmd, &instances); err != nil {
				return err
			}
		} else {
			if err := cli.HandlePayloadStdinSliceWithRuntime[metadata_service.InstanceInfoSpec](rt, cmd, &instances); err != nil {
				return err
			}
		}

		// Send off requests
		results = metadataServiceClient.AddInstanceInfoSpecs(cmd.Context(), rt.Token, instances)
	}

	// Deal with per-request errors
	var reqErrorsOccurred = false
	for _, err := range results.Errors() {
		if err != nil {
			rt.Logger.Error().Err(err).Msg("failed to add instance info")
			reqErrorsOccurred = true
		}
	}
	var names []string
	for _, instance := range results.Values() {
		names = append(names, instance.Metadata.Name)
	}
	rt.Logger.Debug().Msgf("instance infos created: %q", names)
	if reqErrorsOccurred {
		return cli.Errorf(cli.CodeHTTP, "instance info addition completed with errors")
	}

	return nil
}

func newCmdMetadataInstanceAdd() *cobra.Command {
	// metadataInstanceAddCmd represents the "metadata instance add" command
	var metadataInstanceAddCmd = &cobra.Command{
		Use:   "add",
		Args:  cobra.NoArgs,
		Short: "Add one or more instance infos to metadata-service",
		Long: `Add one or more instance infos to metadata-service.

See ochami-metadata(1) for more details.`,
		Example: `  # Add instance info using JSON
  ochami metadata instance add -d \
    '{
       "name": "x1000c0s0b0n0-instance",
       "instance_id": "x1000c0s0b0n0",
       "hostname": "nid001000.demo.cluster",
       "local_hostname": "nid001000",
       "public_keys": ["ssh-ed25519 AAAAC3Nza... admin@demo"]
     }'

  # Add multiple instance infos using JSON array of specs
  ochami metadata instance add -d \
    '[
       {
         "name": "x1000c0s0b0n0-instance",
         "instance_id": "x1000c0s0b0n0"
       },
       {
         "name": "x1000c0s0b0n1-instance",
         "instance_id": "x1000c0s0b0n1"
       }
     ]'

  # Add multiple instance infos using YAML array of specs
  ochami metadata instance add -f yaml <<'EOF'
   - name: x1000c0s0b0n0-instance
     instance_id: "x1000c0s0b0n0"
   - name: x1000c0s0b0n1-instance
     instance_id: "x1000c0s0b0n1"
   EOF

  # Add instance info preserving labels/annotations (envelope API)
  ochami metadata instance add -e -d \
    '{
       "metadata": {
         "name": "x1000c0s0b0n0-instance",
         "labels": {
           "env": "prod"
         }
       },
       "spec": {
         "instance_id": "x1000c0s0b0n0"
       }
     }'

  # Add multiple instances from file
  ochami metadata instance add -d @instances.json
  ochami metadata instance add -d @instance.yaml -f yaml

  # Add instances using data from stdin
  echo '<json_data>' | ochami metadata instance add -d @-
  echo '<yaml_data>' | ochami metadata instance add -d @- -f yaml`,
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
			opts := &metadataInstanceAddOptions{}
			if cmd.Flag("envelope").Changed {
				opts.Envelope, _ = cmd.Flags().GetBool("envelope") //nolint:errcheck // Flag registered with matching type, error impossible
			}

			return runCoreMetadataInstanceAdd(cmd, opts, metadataServiceClient, rt)
		},
	}

	// Create flags
	metadataInstanceAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")

	cli.AddFormatInputFlag(metadataInstanceAddCmd)
	metadataInstanceAddCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return metadataInstanceAddCmd
}
