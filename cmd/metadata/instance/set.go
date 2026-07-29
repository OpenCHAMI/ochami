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
)

func newCmdMetadataInstanceSet() *cobra.Command {
	// metadataInstanceSetCmd represents the "metadata instance set" command
	var metadataInstanceSetCmd = &cobra.Command{
		Use:   "set <uid>",
		Args:  cobra.ExactArgs(1),
		Short: "Set details of an existing instance info spec",
		Long: `Set details of an existing instance info spec.

See ochami-metadata(1) for more details.`,
		Example: `  # Set instance info details using payload data
  ochami metadata instance set instanceinfo-d614b918 -d \
    '{
       "instance_id": "x1000c0s0b0n0",
       "hostname": "nid001000.demo.cluster",
       "local_hostname": "nid001000"
     }'

  # Set instance info details preserving labels/annotations (envelope API)
  ochami metadata instance set instanceinfo-d614b918 -e -d \
    '{
       "metadata": {
         "labels": {
           "env": "prod"
         }
       },
       "spec": {
         "instance_id": "x1000c0s0b0n0"
       }
     }'

  # Set instance info details using file
  ochami metadata instance set instanceinfo-d614b918 -d @instance.json
  ochami metadata instance set instanceinfo-d614b918 -d @instance.yaml -f yaml

  # Set instance info details using data from stdin
  echo '<json_data>' | ochami metadata instance set instanceinfo-d614b918 -d @-
  echo '<json_data>' | ochami metadata instance set instanceinfo-d614b918
  echo '<yaml_data>' | ochami metadata instance set instanceinfo-d614b918 -f yaml -d @-
  echo '<yaml_data>' | ochami metadata instance set instanceinfo-d614b918 -f yaml`,
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

			var instanceSet *api.InstanceInfo
			var reqErr error
			if envelope {
				// Use advanced API (spec, metadata, annotations)

				// Read instance data
				instance := metadata_service_client.UpdateInstanceInfoRequest{}
				if cmd.Flag("data").Changed {
					cli.HandlePayload(cmd, &instance)
				} else {
					cli.HandlePayloadStdin(cmd, &instance)
				}

				// Send off request
				instanceSet, reqErr = metadataServiceClient.SetInstanceInfo(cli.Token, args[0], instance)
			} else {
				// Use simple API (spec)

				// Read instance data
				spec := api.InstanceInfoSpec{}
				if cmd.Flag("data").Changed {
					cli.HandlePayload(cmd, &spec)
				} else {
					cli.HandlePayloadStdin(cmd, &spec)
				}

				// Send off request
				instanceSet, reqErr = metadataServiceClient.SetInstanceInfoSpec(cli.Token, args[0], spec)
			}
			if reqErr != nil {
				log.Logger.Error().Err(reqErr).Msg("failed to set instance info")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Check that a modified item was returned
			if instanceSet == nil {
				log.Logger.Error().Msg("instance info set returned no resource")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Print UIDs of modified items
			log.Logger.Info().Msgf("Instance infos set: %+v", []string{instanceSet.Metadata.UID})
		},
	}

	// Create flags
	metadataInstanceSetCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	metadataInstanceSetCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	metadataInstanceSetCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return metadataInstanceSetCmd
}
