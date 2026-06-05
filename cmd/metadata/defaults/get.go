// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package defaults

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	metadata_service_lib "github.com/OpenCHAMI/ochami/internal/cli/metadata_service"
	"github.com/OpenCHAMI/ochami/internal/log"
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
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			metadataServiceClient := metadata_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			uid := args[0]

			// Make request
			outBytes, err := metadataServiceClient.GetDefaults(cli.Token, cli.FormatOutput, uid)
			if err != nil {
				log.Logger.Error().Err(err).Msgf("failed to get cluster defaults info for %s", uid)
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Print output
			fmt.Print(string(outBytes))
		},
	}

	// Create flags
	metadataDefaultsGetCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	metadataDefaultsGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return metadataDefaultsGetCmd
}
