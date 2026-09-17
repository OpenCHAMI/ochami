// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package defaults

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
	"github.com/openchami/ochami/pkg/client"
)

func newCmdMetadataDefaultsGet() *cobra.Command {
	// metadataDefaultsGetCmd represents the "metadata defaults get" command
	var metadataDefaultsGetCmd = &cobra.Command{
		Use:   "get <uid>",
		Args:  cobra.ExactArgs(1),
		Short: "Get a cluster defaults by its UID",
		Long: `Get a cluster defaults by its UID.

See ochami-metadata(1) for more details.`,
		Example: `  # Get info about a cluster defaults
   ochami metadata defaults get clusterdefaults-773d99bf`,
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
			outBytes, err := metadataServiceClient.GetDefaults(cmd.Context(), cli.Token, cli.FormatOutput, uid)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					return cli.Errorf(cli.CodeHTTP, "failed to get cluster defaults info for %s: %w", uid, err)
				}
				return cli.Errorf(cli.CodeNetwork, "failed to get cluster defaults info for %s: %w", uid, err)
			}

			// Print output
			fmt.Fprint(cli.Ios.Out(), string(outBytes))

			return nil
		},
	}

	// Create flags
	metadataDefaultsGetCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	metadataDefaultsGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return metadataDefaultsGetCmd
}
