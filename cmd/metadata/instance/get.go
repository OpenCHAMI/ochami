// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package instance

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
)

func newCmdMetadataInstanceGet() *cobra.Command {
	// metadataInstanceGetCmd represents the "metadata instance get" command
	var metadataInstanceGetCmd = &cobra.Command{
		Use:   "get <uid>",
		Args:  cobra.ExactArgs(1),
		Short: "Get an instance info by its UID",
		Long: `Get an instance info by its UID.

See ochami-metadata(1) for more details.`,
		Example: `  # Get info about an instance
  ochami metadata instance get instanceinfo-773d99bf

  # Get instance info in YAML format
  ochami metadata instance get instanceinfo-773d99bf -F yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Create client to use for requests
			metadataServiceClient, err := metadata_service_lib.GetClient(cmd, rt)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := rt.HandleToken(cmd); err != nil {
				return err
			}

			uid := args[0]

			// Make request
			outBytes, err := metadataServiceClient.GetInstanceInfo(cmd.Context(), rt.Token, rt.FormatOutput, uid)
			if err != nil {
				return cli.ClassifyClientError(err, fmt.Sprintf("failed to get instance info for %s", uid), fmt.Sprintf("failed to get instance info for %s", uid))
			}

			// Print output
			if err := cli.WriteOutput(rt.Ios.Out(), outBytes); err != nil {
				return err
			}

			return nil
		},
	}

	// Create flags

	cli.AddFormatOutputFlag(metadataInstanceGetCmd)
	metadataInstanceGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return metadataInstanceGetCmd
}
