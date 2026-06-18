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
       "metadata": {
         "name": "demo-cluster-defaults"
       },
       "spec": {
         "base_url": "https://demo.openchami.cluster:8443/cloud-init",
         "cluster_name": "demo",
         "description": "Demo cluster defaults",
         "short_name": "nid",
         "nid_length": 4
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

			// Read cluster defaults data
			defaults := metadata_service_client.UpdateClusterDefaultsRequest{}
			cli.HandlePayload(cmd, &defaults)

			// Send off requests
			defaultsSet, err := metadataServiceClient.SetDefaults(cli.Token, args[0], defaults)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to set cluster defaults")
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
