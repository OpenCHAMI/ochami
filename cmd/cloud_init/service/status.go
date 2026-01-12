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

func newCmdServiceStatus() *cobra.Command {
	// serviceStatusCmd represents the "cloud-init service status" command
	var serviceStatusCmd = &cobra.Command{
		Use:   "status",
		Args:  cobra.NoArgs,
		Short: "Display status of the cloud-init metadata service",
		Long: `Display status of the cloud-init metadata service.

See ochami-cloud-init(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			cloudInitClient := cloud_init_lib.GetClient(cmd)

			if !cmd.Flag("api").Changed {
				if _, err := cloudInitClient.GetVersion(); err != nil {
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(err).Msg("cloud-init status request yielded unsuccessful HTTP response")
						if !cmd.Flag("quiet").Changed {
							fmt.Println("cloud-init is running, but not normally")
						}
						os.Exit(1)
					} else {
						log.Logger.Error().Err(err).Msg("failed to get cloud-init status")
						if !cmd.Flag("quiet").Changed {
							fmt.Println("cloud-init is not running")
						}
						os.Exit(1)
					}
				} else {
					if !cmd.Flag("quiet").Changed {
						fmt.Println("cloud-init is running")
					}
					os.Exit(0)
				}
			}

			var respArr []client.HTTPEnvelope
			errOccurred := false
			if cmd.Flag("api").Changed {
				if henv, err := cloudInitClient.GetAPI(); err != nil {
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(err).Msg("cloud-init API spec request yielded unsuccessful HTTP response")
					} else {
						log.Logger.Error().Err(err).Msg("failed to get cloud-init API spec")
					}
					errOccurred = true
				} else {
					respArr = append(respArr, henv)
				}
			}

			for _, henv := range respArr {
				if outBytes, err := client.FormatBody(henv.Body, cli.FormatOutput); err != nil {
					log.Logger.Error().Err(err).Msg("failed to format output")
					cli.LogHelpError(cmd)
					os.Exit(1)
				} else {
					fmt.Print(string(outBytes))
				}
			}

			if errOccurred {
				log.Logger.Warn().Msg("one or more requests to cloud-init failed")
				os.Exit(1)
			}
		},
	}

	// Create flags
	serviceStatusCmd.Flags().Bool("api", false, "print OpenAPI spec")
	serviceStatusCmd.Flags().BoolP("quiet", "q", false, "don't print output; return 0 if running, 1 if not")
	serviceStatusCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	serviceStatusCmd.MarkFlagsMutuallyExclusive("quiet", "api")

	serviceStatusCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return serviceStatusCmd
}
