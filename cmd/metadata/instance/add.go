// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package instance

import (
	"os"

	metadata_service_client "github.com/OpenCHAMI/metadata-service/pkg/client"
	"github.com/spf13/cobra"

	api "github.com/OpenCHAMI/metadata-service/apis/cloud-init.openchami.io/v1"

	"github.com/OpenCHAMI/ochami/internal/cli"
	metadata_service_lib "github.com/OpenCHAMI/ochami/internal/cli/metadata_service"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client/metadata_service"
)

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
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			metadataServiceClient := metadata_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Determine how to read payload (simple versus advanced API)
			envelope, flagErr := cmd.Flags().GetBool("envelope")
			if flagErr != nil {
				log.Logger.Warn().Err(flagErr).Msg("failed to read --envelope, falling back to simple API")
			}

			var instancesCreated []api.InstanceInfo
			var reqErrs []error
			var reqErr error
			if envelope {
				// Use advanced API (spec, metadata, annotations)

				// Read instance data
				instances := []metadata_service_client.CreateInstanceInfoRequest{}
				if cmd.Flag("data").Changed {
					cli.HandlePayloadSlice[metadata_service_client.CreateInstanceInfoRequest](cmd, &instances)
				} else {
					cli.HandlePayloadStdinSlice[metadata_service_client.CreateInstanceInfoRequest](cmd, &instances)
				}

				// Send off requests
				instancesCreated, reqErrs, reqErr = metadataServiceClient.AddInstanceInfos(cli.Token, instances)
			} else {
				// Use simple API (spec)

				// Read instance data
				instances := []metadata_service.InstanceInfoSpec{}
				if cmd.Flag("data").Changed {
					cli.HandlePayloadSlice[metadata_service.InstanceInfoSpec](cmd, &instances)
				} else {
					cli.HandlePayloadStdinSlice[metadata_service.InstanceInfoSpec](cmd, &instances)
				}

				// Send off requests
				instancesCreated, reqErrs, reqErr = metadataServiceClient.AddInstanceInfoSpecs(cli.Token, instances)
			}

			// Handle any non-request error
			if reqErr != nil {
				log.Logger.Error().Err(reqErr).Msg("failed to add instance infos")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Deal with per-request errors
			var reqErrorsOccurred = false
			for _, err := range reqErrs {
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to add instance info")
					reqErrorsOccurred = true
				}
			}

			// Print UIDs of created items
			var uids []string
			for _, instance := range instancesCreated {
				uids = append(uids, instance.Metadata.UID)
			}
			log.Logger.Info().Msgf("Instance infos created: %+v", uids)

			// Warn if any request errors occurred
			if reqErrorsOccurred {
				cli.LogHelpError(cmd)
				log.Logger.Warn().Msg("Instance info addition completed with errors")
				os.Exit(1)
			}
		},
	}

	// Create flags
	metadataInstanceAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	metadataInstanceAddCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	metadataInstanceAddCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return metadataInstanceAddCmd
}
