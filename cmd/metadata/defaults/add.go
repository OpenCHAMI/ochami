// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package defaults

import (
	"os"

	metadata_service_client "github.com/OpenCHAMI/metadata-service/pkg/client"
	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	metadata_service_lib "github.com/OpenCHAMI/ochami/internal/cli/metadata_service"
	"github.com/OpenCHAMI/ochami/internal/log"
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
         "base_url": "https://demo1.openchami.cluster:8443/cloud-init",
         "cluster_name": "demo1",
         "description": "Demo 1 cluster defaults",
         "short_name": "nid",
         "nid_length": 4
       },
       {
         "base_url": "https://demo2.openchami.cluster:8443/cloud-init",
         "cluster_name": "demo2",
         "description": "Demo 2 cluster defaults",
         "short_name": "de",
         "nid_length": 3
       }
     ]'

  # Add multiple cluster defaults using YAML array
  ochami metadata defaults add -f yaml <<'EOF'
  - base_url: "https://demo1.openchami.cluster:8443/cloud-init"
    cluster_name: "demo1"
  - base_url: "https://demo2.openchami.cluster:8443/cloud-init"
    cluster_name: "demo2"
  EOF

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

			// Read node data
			defaults := []metadata_service_client.CreateClusterDefaultsRequest{}
			if cmd.Flag("data").Changed {
				cli.HandlePayloadSlice[metadata_service_client.CreateClusterDefaultsRequest](cmd, &defaults)
			} else {
				cli.HandlePayloadStdinSlice[metadata_service_client.CreateClusterDefaultsRequest](cmd, &defaults)
			}

			// Send off requests
			defaultsCreated, errs, err := metadataServiceClient.AddDefaults(cli.Token, defaults)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to add cluster defaults")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Deal with per-request errors
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to add cluster defaults")
					errorsOccurred = true
				}
			}
			log.Logger.Debug().Msgf("Cluster defaults created: %+v", defaultsCreated)
			if errorsOccurred {
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
