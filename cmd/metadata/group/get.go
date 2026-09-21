// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
)

func newCmdMetadataGroupGet() *cobra.Command {
	// metadataGroupGetCmd represents the "metadata group get" command
	var metadataGroupGetCmd = &cobra.Command{
		Use:   "get <uid>",
		Args:  cobra.ExactArgs(1),
		Short: "Get a group by its UID",
		Long: `Get a group by its UID.

See ochami-metadata(1) for more details.`,
		Example: `  # Get info about a group
  ochami metadata group get group-773d99bf

  # Get group in YAML format
  ochami metadata group get group-773d99bf -F yaml`,
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

			uid := args[0]

			// Make request
			outBytes, err := metadataServiceClient.GetGroup(cmd.Context(), cli.Token, cli.FormatOutput, uid)
			if err != nil {
				return cli.ClassifyClientError(err, fmt.Sprintf("failed to get group info for %s", uid), fmt.Sprintf("failed to get group info for %s", uid))
			}

			// Print output
			fmt.Fprint(cli.Ios.Out(), string(outBytes))

			return nil
		},
	}

	// Create flags
	metadataGroupGetCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	metadataGroupGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return metadataGroupGetCmd
}
