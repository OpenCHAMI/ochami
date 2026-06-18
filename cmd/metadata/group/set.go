// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"os"

	metadata_service_client "github.com/OpenCHAMI/metadata-service/pkg/client"
	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	metadata_service_lib "github.com/OpenCHAMI/ochami/internal/cli/metadata_service"
	"github.com/OpenCHAMI/ochami/internal/log"
)

func newCmdMetadataGroupSet() *cobra.Command {
	// metadataGroupSetCmd represents the "metadata group set" command
	var metadataGroupSetCmd = &cobra.Command{
		Use:   "set <uid>",
		Args:  cobra.ExactArgs(1),
		Short: "Set details of an existing group spec",
		Long: `Set details of an existing group spec.

See ochami-metadata(1) for more details.`,
		Example: `  # Set group details using payload data
  ochami metadata group set group-d614b918 -d \
    '{
       "metadata": {
         "name": "compute-group"
       },
       "spec": {
         "template":"#cloud-config\npackages:\n  - vim\n",
         "metaData":{"role":"compute"},
         "osVersion":"ubuntu-22.04"
       }
     }'

  # Set group details using file
  ochami metadata group set group-d614b918 -d @group.json
  ochami metadata group set group-d614b918 -d @group.yaml -f yaml

  # Set group details using data from stdin
  echo '<json_data>' | ochami metadata group set group-d614b918 -d @-
  echo '<json_data>' | ochami metadata group set group-d614b918
  echo '<yaml_data>' | ochami metadata group set group-d614b918 -f yaml -d @-
  echo '<yaml_data>' | ochami metadata group set group-d614b918 -f yaml`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			metadataServiceClient := metadata_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Read group data
			group := metadata_service_client.UpdateGroupRequest{}
			cli.HandlePayload(cmd, &group)

			// Send off requests
			groupSet, err := metadataServiceClient.SetGroup(cli.Token, args[0], group)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to set group")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			log.Logger.Debug().Msgf("Group set: %+v", groupSet)
		},
	}

	// Create flags
	metadataGroupSetCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	metadataGroupSetCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	metadataGroupSetCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return metadataGroupSetCmd
}
