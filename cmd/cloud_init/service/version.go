// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package service

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	cloud_init_lib "github.com/openchami/ochami/internal/cli/cloud_init"
)

func newCmdServiceVersion() *cobra.Command {
	// serviceVersionCmd represents the "cloud-init service status" command
	var serviceVersionCmd = &cobra.Command{
		Use:   "version",
		Args:  cobra.NoArgs,
		Short: "Print version of the cloud-init metadata service",
		Long: `Print version of the cloud-init metadata service.

See ochami-cloud-init(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			cloudInitClient, err := cloud_init_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			henv, err := cloudInitClient.GetVersion()
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					return cli.Errorf(cli.CodeHTTP, "cloud-init version request yielded unsuccessful HTTP response: %w", err)
				}
				return cli.Errorf(cli.CodeNetwork, "failed to get cloud-init version: %w", err)
			}

			outBytes, err := client.FormatBody(henv.Body, cli.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
			}
			fmt.Print(string(outBytes))

			return nil
		},
	}

	// Create flags
	serviceVersionCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	serviceVersionCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return serviceVersionCmd
}
