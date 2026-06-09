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
       "template":"#cloud-config\npackages:\n  - vim\n",
       "metaData":{"role":"storage"}
     }'

  # Add multiple groups using JSON array
  ochami metadata group add -d \
    '[
       {
         "template":"#cloud-config\npackages:\n  - nfs-common\n"
       },
       {
         "template":"#cloud-config\npackages:\n  - nfs-server\n"
       }
     ]'

  # Add multiple groups using YAML array
  ochami metadata group add -f yaml <<'EOF'
  - template: |
      #cloud-config
      packages:
        - nfs-common
  - template: |
      #cloud-config
      packages:
        - nfs-server
  EOF

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

			// Read group data
			groups := []metadata_service_client.CreateGroupRequest{}
			if cmd.Flag("data").Changed {
				cli.HandlePayloadSlice[metadata_service_client.CreateGroupRequest](cmd, &groups)
			} else {
				cli.HandlePayloadStdinSlice[metadata_service_client.CreateGroupRequest](cmd, &groups)
			}

			// Send off requests
			groupsCreated, errs, err := metadataServiceClient.AddGroups(cli.Token, groups)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to add groups")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Deal with per-request errors
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to add group")
					errorsOccurred = true
				}
			}
			log.Logger.Debug().Msgf("Groups created: %+v", groupsCreated)
			if errorsOccurred {
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
