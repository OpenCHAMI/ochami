// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	metadata_service_client "github.com/openchami/metadata-service/pkg/client"
	"github.com/spf13/cobra"

	api "github.com/openchami/metadata-service/apis/cloud-init.openchami.io/v1"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client/metadata_service"

	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
)

// metadataGroupSetOptions holds the flag values for the metadata group set command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type metadataGroupSetOptions struct {
	Envelope bool
}

// runCoreMetadataGroupSet contains the core logic for the metadata group set command.
// It takes the parsed options and performs the actual work of setting group details.
func runCoreMetadataGroupSet(cmd *cobra.Command, opts *metadataGroupSetOptions, args []string, metadataServiceClient *metadata_service.MetadataServiceClient) error {
	// Handle token for this command
	if err := cli.HandleToken(cmd); err != nil {
		return err
	}

	// Determine how to read payload (simple versus advanced API)
	var groupSet *api.Group
	var reqErr error
	if opts.Envelope {
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
		groupSet, reqErr = metadataServiceClient.SetGroup(cmd.Context(), cli.Token, args[0], group)
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
		groupSet, reqErr = metadataServiceClient.SetGroupSpec(cmd.Context(), cli.Token, args[0], spec)
	}
	if reqErr != nil {
		return cli.ClassifyClientError(reqErr, "failed to set group", "failed to set group")

	}

	// Check that a modified item was returned
	if groupSet == nil {
		return cli.Errorf(cli.CodeGeneric, "group set returned no resource")
	}

	log.Logger.Debug().Msgf("group set: %+v", groupSet)

	return nil
}

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

			// Extract options from flags
			// Since flags are registered with the correct types on this command,
			// these Get* calls cannot fail, so we ignore errors with explicit comments
			opts := &metadataGroupSetOptions{}
			if cmd.Flag("envelope").Changed {
				opts.Envelope, _ = cmd.Flags().GetBool("envelope") //nolint:errcheck // Flag registered with matching type, error impossible
			}

			return runCoreMetadataGroupSet(cmd, opts, args, metadataServiceClient)
		},
	}

	// Create flags
	metadataGroupSetCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	metadataGroupSetCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	metadataGroupSetCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return metadataGroupSetCmd
}
