// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

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
   EOF

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
   EOF

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

			var groupsCreated []api.Group
			var reqErrs []error
			var reqErr error
			if envelope {
				// Use advanced API (spec, metadata, annotations)

				// Read group data
				groups := []metadata_service_client.CreateGroupRequest{}
				if cmd.Flag("data").Changed {
					cli.HandlePayloadSlice[metadata_service_client.CreateGroupRequest](cmd, &groups)
				} else {
					cli.HandlePayloadStdinSlice[metadata_service_client.CreateGroupRequest](cmd, &groups)
				}

				// Send off requests
				groupsCreated, reqErrs, reqErr = metadataServiceClient.AddGroups(cli.Token, groups)
			} else {
				// Use simple API (spec)

				// Read group data
				groups := []metadata_service.GroupSpec{}
				if cmd.Flag("data").Changed {
					cli.HandlePayloadSlice[metadata_service.GroupSpec](cmd, &groups)
				} else {
					cli.HandlePayloadStdinSlice[metadata_service.GroupSpec](cmd, &groups)
				}

				// Send off requests
				groupsCreated, reqErrs, reqErr = metadataServiceClient.AddGroupSpecs(cli.Token, groups)
			}

			// Handle any non-request error
			if reqErr != nil {
				log.Logger.Error().Err(reqErr).Msg("failed to add groups")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Deal with per-request errors
			var reqErrorsOccurred = false
			for _, err := range reqErrs {
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to add group")
					reqErrorsOccurred = true
				}
			}

			// Print UIDs of created items
			var uids []string
			for _, group := range groupsCreated {
				uids = append(uids, group.Metadata.UID)
			}
			log.Logger.Info().Msgf("Groups created: %+v", uids)

			// Warn if any request errors occurred
			if reqErrorsOccurred {
				cli.LogHelpError(cmd)
				log.Logger.Warn().Msg("Group addition completed with errors")
				os.Exit(1)
			}
		},
	}

	// Create flags
	metadataGroupAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	metadataGroupAddCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	metadataGroupAddCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return metadataGroupAddCmd
}
