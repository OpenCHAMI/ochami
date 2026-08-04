// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package defaults

import (
	"os"

	metadata_service_client "github.com/openchami/metadata-service/pkg/client"
	"github.com/spf13/cobra"

	api "github.com/openchami/metadata-service/apis/cloud-init.openchami.io/v1"

	"github.com/OpenCHAMI/ochami/internal/cli"
	metadata_service_lib "github.com/OpenCHAMI/ochami/internal/cli/metadata_service"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client/metadata_service"
)

func newCmdMetadataDefaultsAdd() *cobra.Command {
	// metadataDefaultsAddCmd represents the "metadata defaults add" command
	var metadataDefaultsAddCmd = &cobra.Command{
		Use:   "add",
		Args:  cobra.NoArgs,
		Short: "Add one or more cluster defaults to metadata-service",
		Long: `Add one or more cluster defaults to metadata-service.

See ochami-metadata(1) for more details.`,
		Example: `  # Add cluster defaults using payload data
  ochami metadata defaults add -d \
    '{
       "name": "demo-cluster-defaults",
       "base_url": "https://demo.openchami.cluster:8443/cloud-init",
       "cluster_name": "demo",
       "description": "Demo cluster defaults",
       "short_name": "nid",
       "nid_length": 4
     }'

  # Add multiple cluster defaults using payload data
  ochami metadata defaults add -d \
    '[
       {
         "name": "demo1-cluster-defaults",
         "base_url": "https://demo1.openchami.cluster:8443/cloud-init",
         "cluster_name": "demo1",
         "description": "Demo 1 cluster defaults",
         "short_name": "nid",
         "nid_length": 4
       },
       {
         "name": "demo2-cluster-defaults",
         "base_url": "https://demo2.openchami.cluster:8443/cloud-init",
         "cluster_name": "demo2",
         "description": "Demo 2 cluster defaults",
         "short_name": "de",
         "nid_length": 3
       }
     ]'

  # Add multiple cluster defaults using YAML array of specs
  ochami metadata defaults add -f yaml <<'EOF'
   - name: demo1-cluster-defaults
     base_url: "https://demo1.openchami.cluster:8443/cloud-init"
     cluster_name: "demo1"
   - name: demo2-cluster-defaults
     base_url: "https://demo2.openchami.cluster:8443/cloud-init"
     cluster_name: "demo2"
   EOF

  # Add cluster defaults preserving labels/annotations (envelope API)
  ochami metadata defaults add -e -d \
    '{
       "metadata": {
         "name": "demo-cluster-defaults",
         "labels": {
           "env": "prod"
         }
       },
       "spec": {
         "base_url": "https://demo.openchami.cluster:8443/cloud-init",
         "cluster_name": "demo"
       }
     }'

  # Add cluster defaults using input payload file
  ochami metadata defaults add -d @payload.json
  ochami metadata defaults add -d @payload.yaml -f yaml

  # Add cluster defaults using data from stdin
  echo '<json_data>' | ochami metadata defaults add -d @-
  echo '<json_data>' | ochami metadata defaults add
  echo '<yaml_data>' | ochami metadata defaults add -f yaml -d @-
  echo '<yaml_data>' | ochami metadata defaults add -f yaml`,
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

			var defaultsCreated []api.ClusterDefaults
			var reqErrs []error
			var reqErr error
			if envelope {
				// Use advanced API (spec, metadata, annotations)

				// Read cluster defaults data
				defaults := []metadata_service_client.CreateClusterDefaultsRequest{}
				if cmd.Flag("data").Changed {
					cli.HandlePayloadSlice[metadata_service_client.CreateClusterDefaultsRequest](cmd, &defaults)
				} else {
					cli.HandlePayloadStdinSlice[metadata_service_client.CreateClusterDefaultsRequest](cmd, &defaults)
				}

				// Send off requests
				defaultsCreated, reqErrs, reqErr = metadataServiceClient.AddDefaults(cli.Token, defaults)
			} else {
				// Use simple API (spec)

				// Read cluster defaults data
				defaults := []metadata_service.ClusterDefaultsSpec{}
				if cmd.Flag("data").Changed {
					cli.HandlePayloadSlice[metadata_service.ClusterDefaultsSpec](cmd, &defaults)
				} else {
					cli.HandlePayloadStdinSlice[metadata_service.ClusterDefaultsSpec](cmd, &defaults)
				}

				// Send off requests
				defaultsCreated, reqErrs, reqErr = metadataServiceClient.AddDefaultsSpecs(cli.Token, defaults)
			}

			// Handle any non-request error
			if reqErr != nil {
				log.Logger.Error().Err(reqErr).Msg("failed to add cluster defaults")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Deal with per-request errors
			var reqErrorsOccurred = false
			for _, err := range reqErrs {
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to add cluster defaults")
					reqErrorsOccurred = true
				}
			}

			// Print UIDs of created items
			var uids []string
			for _, defaults := range defaultsCreated {
				uids = append(uids, defaults.Metadata.UID)
			}
			log.Logger.Info().Msgf("Cluster defaults created: %+v", uids)

			// Warn if any request errors occurred
			if reqErrorsOccurred {
				cli.LogHelpError(cmd)
				log.Logger.Warn().Msg("Cluster defaults addition completed with errors")
				os.Exit(1)
			}
		},
	}

	// Create flags
	metadataDefaultsAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	metadataDefaultsAddCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	metadataDefaultsAddCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return metadataDefaultsAddCmd
}
