// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	metadata_service_client "github.com/openchami/metadata-service/pkg/client"
	"github.com/spf13/cobra"

	api "github.com/openchami/metadata-service/apis/cloud-init.openchami.io/v1"

	"github.com/openchami/ochami/internal/cli"
	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
	"github.com/openchami/ochami/internal/log"
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
        "template":"#cloud-config\npackages:\n  - vim\n",
        "metaData":{"role":"compute"},
        "osVersion":"ubuntu-22.04"
      }'

  # Set group details preserving labels/annotations (envelope API)
  ochami metadata group set group-d614b918 -e -d \
     '{
        "metadata": {
          "labels": {
            "role": "compute"
          }
        },
        "spec": {
          "template":"#cloud-config\npackages:\n  - vim\n"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			metadataServiceClient, err := metadata_service_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// Determine how to read payload (simple versus advanced API)
			envelope, _ := cmd.Flags().GetBool("envelope") //nolint:errcheck // flag is registered with the matching type on this command

			var groupSet *api.Group
			var reqErr error
			if envelope {
				// Use advanced API (spec, metadata, annotations)

				// Read group data
				group := metadata_service_client.UpdateGroupRequest{}
				if cmd.Flag("data").Changed {
					if err := cli.HandlePayload(cmd, &group); err != nil {
						return err
					}
				} else {
					if err := cli.HandlePayloadStdin(cmd, &group); err != nil {
						return err
					}
				}

				// Send off request
				groupSet, reqErr = metadataServiceClient.SetGroup(cli.Token, args[0], group)
			} else {
				// Use simple API (spec)

				// Read group data
				spec := api.GroupSpec{}
				if cmd.Flag("data").Changed {
					if err := cli.HandlePayload(cmd, &spec); err != nil {
						return err
					}
				} else {
					if err := cli.HandlePayloadStdin(cmd, &spec); err != nil {
						return err
					}
				}

				// Send off request
				groupSet, reqErr = metadataServiceClient.SetGroupSpec(cli.Token, args[0], spec)
			}
			if reqErr != nil {
				return cli.ClassifyClientError(reqErr, "failed to set group", "failed to set group")

			}

			// Check that a modified item was returned
			if groupSet == nil {
				return cli.Errorf(cli.CodeGeneric, "group set returned no resource")
			}

			// Print UIDs of modified items
			log.Logger.Info().Msgf("Groups set: %+v", []string{groupSet.Metadata.UID})

			return nil
		},
	}

	// Create flags
	metadataGroupSetCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	metadataGroupSetCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	metadataGroupSetCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return metadataGroupSetCmd
}
