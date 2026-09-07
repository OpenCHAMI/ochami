// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"errors"

	metadata_service_client "github.com/openchami/metadata-service/pkg/client"
	"github.com/spf13/cobra"

	api "github.com/openchami/metadata-service/apis/cloud-init.openchami.io/v1"

	"github.com/openchami/ochami/internal/cli"
	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/metadata_service"
)

func newCmdMetadataGroupAdd() *cobra.Command {
	// metadataGroupAddCmd represents the "metadata group add" command
	var metadataGroupAddCmd = &cobra.Command{
		Use:   "add",
		Args:  cobra.NoArgs,
		Short: "Add one or more groups to metadata-service",
		Long: `Add one or more groups to metadata-service.

See ochami-metadata(1) for more details.`,
		Example: `  # Add group with inline multi-line template (YAML via stdin)
   ochami metadata group add -f yaml <<'EOF'
    name: compute-group
    template: |
      #cloud-config
      package_update: true
      packages:
        - nfs-common
        - chrony
    metaData:
      role: compute
    EOF'

  # Add group using JSON (single line template)
  ochami metadata group add -d \
     '{
        "name": "storage-group",
        "template":"#cloud-config\npackages:\n  - vim\n",
        "metaData":{"role":"storage"}
      }'

  # Add multiple groups using JSON array of specs
  ochami metadata group add -d \
     '[
        {
          "name": "nfs-client-group",
          "template":"#cloud-config\npackages:\n  - nfs-common\n"
        },
        {
          "name": "nfs-server-group",
          "template":"#cloud-config\npackages:\n  - nfs-server\n"
        }
      ]'

  # Add multiple groups using YAML array of specs
  ochami metadata group add -f yaml <<'EOF'
   - name: nfs-client-group
     template: |
       #cloud-config
       packages:
         - nfs-common
   - name: nfs-server-group
     template: |
       #cloud-config
       packages:
         - nfs-server
   EOF'

  # Add group preserving labels/annotations (envelope API)
  ochami metadata group add -e -d \
     '{
        "metadata": {
          "name": "storage-group",
          "labels": {
            "role": "storage"
          }
        },
        "spec": {
          "template":"#cloud-config\npackages:\n  - vim\n"
        }
      }'

  # Add multiple groups from file
  ochami metadata group add -d @groups.json
  ochami metadata group add -d @groups.yaml -f yaml

  # Add groups using data from stdin
  echo '<json_data>' | ochami metadata group add -d @-
  echo '<json_data>' | ochami metadata group add
  echo '<yaml_data>' | ochami metadata group add -f yaml -d @-
  echo '<yaml_data>' | ochami metadata group add -f yaml`,
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
			envelope, _ := cmd.Flags().GetBool("envelope")

			var groupsCreated []api.Group
			var reqErrs []error
			var reqErr error
			if envelope {
				// Use advanced API (spec, metadata, annotations)

				// Read group data
				groups := []metadata_service_client.CreateGroupRequest{}
				if cmd.Flag("data").Changed {
					if err := cli.HandlePayloadSlice[metadata_service_client.CreateGroupRequest](cmd, &groups); err != nil {
						return err
					}
				} else {
					if err := cli.HandlePayloadStdinSlice[metadata_service_client.CreateGroupRequest](cmd, &groups); err != nil {
						return err
					}
				}

				// Send off requests
				groupsCreated, reqErrs, reqErr = metadataServiceClient.AddGroups(cli.Token, groups)
			} else {
				// Use simple API (spec)

				// Read group data
				groups := []metadata_service.GroupSpec{}
				if cmd.Flag("data").Changed {
					if err := cli.HandlePayloadSlice[metadata_service.GroupSpec](cmd, &groups); err != nil {
						return err
					}
				} else {
					if err := cli.HandlePayloadStdinSlice[metadata_service.GroupSpec](cmd, &groups); err != nil {
						return err
					}
				}

				// Send off requests
				groupsCreated, reqErrs, reqErr = metadataServiceClient.AddGroupSpecs(cli.Token, groups)
			}

			// Handle any non-request error
			if reqErr != nil {
				if errors.Is(reqErr, client.UnsuccessfulHTTPError) {
					return cli.Errorf(cli.CodeHTTP, "failed to add groups: %w", reqErr)
				}
				return cli.Errorf(cli.CodeNetwork, "failed to add groups: %w", reqErr)
			}

			// Deal with per-request errors
			var reqErrorsOccurred = false
			for _, err := range reqErrs {
				if err != nil {
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(err).Msg("failed to add group")
					} else {
						log.Logger.Error().Err(err).Msg("failed to add group")
					}
					reqErrorsOccurred = true
				}
			}

			// Print names of created items
			var names []string
			for _, group := range groupsCreated {
				names = append(names, group.Metadata.Name)
			}
			log.Logger.Info().Msgf("Groups created: %q", names)

			// Warn if any request errors occurred
			if reqErrorsOccurred {
				return cli.Errorf(cli.CodeHTTP, "Group addition completed with errors")
			}

			return nil
		},
	}

	// Create flags
	metadataGroupAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	metadataGroupAddCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	metadataGroupAddCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return metadataGroupAddCmd
}
