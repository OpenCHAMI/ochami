// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package defaults

import (
	"os"

	metadata_service_client "github.com/openchami/metadata-service/pkg/client"
	"github.com/spf13/cobra"

	api "github.com/openchami/metadata-service/apis/cloud-init.openchami.io/v1"

	"github.com/openchami/ochami/internal/cli"
	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
	"github.com/openchami/ochami/internal/log"
)

func newCmdMetadataDefaultsSet() *cobra.Command {
	// metadataDefaultsSetCmd represents the "metadata defaults set" command
	var metadataDefaultsSetCmd = &cobra.Command{
		Use:   "set <uid>",
		Args:  cobra.ExactArgs(1),
		Short: "Set details of an existing cluster defaults spec",
		Long: `Set details of an existing cluster defaults spec.

See ochami-metadata(1) for more details.`,
		Example: `  # Set cluster defaults details using payload data
  ochami metadata defaults set clusterdefaults-d614b918 -d \
    '{
       "base_url": "https://demo.openchami.cluster:8443/cloud-init",
       "cluster_name": "demo",
       "description": "Demo cluster defaults",
       "short_name": "nid",
       "nid_length": 4
     }'

  # Set cluster defaults details preserving labels/annotations (envelope API)
  ochami metadata defaults set clusterdefaults-d614b918 -e -d \
    '{
       "metadata": {
         "labels": {
           "env": "prod"
         }
       },
       "spec": {
         "base_url": "https://demo.openchami.cluster:8443/cloud-init",
         "cluster_name": "demo"
       }
     }'

  # Set cluster defaults details using input payload file
  ochami metadata defaults set clusterdefaults-d614b918 -d @payload.json
  ochami metadata defaults set clusterdefaults-d614b918 -d @payload.yaml -f yaml

  # Set cluster defaults details using data from stdin
  echo '<json_data>' | ochami metadata defaults set clusterdefaults-d614b918 -d @-
  echo '<json_data>' | ochami metadata defaults set clusterdefaults-d614b918
  echo '<yaml_data>' | ochami metadata defaults set clusterdefaults-d614b918 -f yaml -d @-
  echo '<yaml_data>' | ochami metadata defaults set clusterdefaults-d614b918 -f yaml`,
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

			var defaultsSet *api.ClusterDefaults
			var reqErr error
			if envelope {
				// Use advanced API (spec, metadata, annotations)

				// Read cluster defaults data
				defaults := metadata_service_client.UpdateClusterDefaultsRequest{}
				if cmd.Flag("data").Changed {
					cli.HandlePayload(cmd, &defaults)
				} else {
					cli.HandlePayloadStdin(cmd, &defaults)
				}

				// Send off request
				defaultsSet, reqErr = metadataServiceClient.SetDefaults(cli.Token, args[0], defaults)
			} else {
				// Use simple API (spec)

				// Read cluster defaults data
				spec := api.ClusterDefaultsSpec{}
				if cmd.Flag("data").Changed {
					cli.HandlePayload(cmd, &spec)
				} else {
					cli.HandlePayloadStdin(cmd, &spec)
				}

				// Send off request
				defaultsSet, reqErr = metadataServiceClient.SetDefaultsSpec(cli.Token, args[0], spec)
			}
			if reqErr != nil {
				log.Logger.Error().Err(reqErr).Msg("failed to set cluster defaults")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Check that a modified item was returned
			if defaultsSet == nil {
				log.Logger.Error().Msg("cluster defaults set returned no resource")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Print UIDs of modified items
			log.Logger.Info().Msgf("Cluster defaults set: %+v", []string{defaultsSet.Metadata.UID})
		},
	}

	// Create flags
	metadataDefaultsSetCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	metadataDefaultsSetCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	metadataDefaultsSetCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return metadataDefaultsSetCmd
}
