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

	bss_lib "github.com/OpenCHAMI/ochami/internal/cli/bss"
)

func newCmdServiceStatus() *cobra.Command {
	// serviceStatusCmd represents the "bss service status" command
	var serviceStatusCmd = &cobra.Command{
		Use:   "status",
		Args:  cobra.NoArgs,
		Short: "Display status of the Boot Script Service (BSS)",
		Long: `Display status of the Boot Script Service (BSS).

See ochami-bss(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			bssClient := bss_lib.GetClient(cmd)

			// Determine which component to get status for and send request
			var httpEnv client.HTTPEnvelope
			var err error
			if cmd.Flag("all").Changed {
				httpEnv, err = bssClient.GetStatus("all")
			} else if cmd.Flag("storage").Changed {
				httpEnv, err = bssClient.GetStatus("storage")
			} else if cmd.Flag("smd").Changed {
				httpEnv, err = bssClient.GetStatus("smd")
			} else {
				httpEnv, err = bssClient.GetStatus("")
			}
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("BSS status request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to get BSS status")
				}
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Print output
			if outBytes, err := client.FormatBody(httpEnv.Body, cli.FormatOutput); err != nil {
				log.Logger.Error().Err(err).Msg("failed to format output")
				cli.LogHelpError(cmd)
				os.Exit(1)
			} else {
				fmt.Print(string(outBytes))
			}
		},
	}

	// Create flags
	serviceStatusCmd.Flags().Bool("all", false, "print all status data from BSS")
	serviceStatusCmd.Flags().Bool("storage", false, "print status of storage backend from BSS")
	serviceStatusCmd.Flags().Bool("smd", false, "print status of BSS connection to SMD")
	serviceStatusCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	serviceStatusCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)
	serviceStatusCmd.MarkFlagsMutuallyExclusive("all", "storage", "smd")

	return serviceStatusCmd
}
