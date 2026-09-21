// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package service

import (
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
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Create client to use for requests
			metadataServiceClient, err := metadata_service_lib.GetClientWithRuntime(cmd, rt)
			if err != nil {
				return err
			}

			// Make request
			outbytes, err := metadataServiceClient.GetHealth(cmd.Context(), rt.FormatOutput)
			if err != nil {
				return cli.ClassifyClientError(err, "failed to get metadata-service health", "failed to get metadata-service health")

			}

			// Print output
			if err := cli.WriteOutput(rt.Ios.Out(), outbytes); err != nil {
				return err
			}

			return nil
		},
	}

	// Create flags

	cli.AddFormatOutputFlag(serviceStatusCmd)
	serviceStatusCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return serviceStatusCmd
}
