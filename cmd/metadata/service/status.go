// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package service

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
)

func newCmdServiceStatus() *cobra.Command {
	// serviceStatusCmd represents the "metadata service status" command
	var serviceStatusCmd = &cobra.Command{
		Use:   "status",
		Args:  cobra.NoArgs,
		Short: "Display status of the metadata service",
		Long: `Display status of the metadata service.

See ochami-metadata(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			metadataServiceClient, err := metadata_service_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Make request
			outbytes, err := metadataServiceClient.GetHealth(cmd.Context(), cli.FormatOutput)
			if err != nil {
				return cli.ClassifyClientError(err, "failed to get metadata-service health", "failed to get metadata-service health")

			}

			// Print output
			fmt.Fprint(cli.Ios.Out(), string(outbytes))

			return nil
		},
	}

	// Create flags
	serviceStatusCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	serviceStatusCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return serviceStatusCmd
}
