// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package service

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"

	cloud_init_lib "github.com/OpenCHAMI/ochami/internal/cli/cloud_init"
)

func newCmdServiceVersion() *cobra.Command {
	// serviceVersionCmd represents the "cloud-init service status" command
	var serviceVersionCmd = &cobra.Command{
		Use:   "version",
		Args:  cobra.NoArgs,
		Short: "Print version of the cloud-init metadata service",
		Long: `Print version of the cloud-init metadata service.

See ochami-cloud-init(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			cloudInitClient := cloud_init_lib.GetClient(cmd)

			henv, err := cloudInitClient.GetVersion()
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("cloud-init version request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to get cloud-init version")
				}
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			if outBytes, err := client.FormatBody(henv.Body, cli.FormatOutput); err != nil {
				log.Logger.Error().Err(err).Msg("failed to format output")
				cli.LogHelpError(cmd)
				os.Exit(1)
			} else {
				fmt.Print(string(outBytes))
			}
		},
	}

	// Create flags
	serviceVersionCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	serviceVersionCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return serviceVersionCmd
}
