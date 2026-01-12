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

func newCmdServiceVersion() *cobra.Command {
	// serviceVersionCmd represents the "bss service version" command
	var serviceVersionCmd = &cobra.Command{
		Use:   "version",
		Args:  cobra.NoArgs,
		Short: "Print version of the Boot Script Service (BSS)",
		Long: `Print version of the Boot Script Service (BSS).

See ochami-bss(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			bssClient := bss_lib.GetClient(cmd)

			// Determine which component to get status for and send request
			httpEnv, err := bssClient.GetStatus("version")
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("BSS version request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to get BSS version")
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

	return serviceVersionCmd
}
