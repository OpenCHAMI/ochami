// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package history

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"

	bss_lib "github.com/OpenCHAMI/ochami/internal/cli/bss"
)

func NewCmd() *cobra.Command {
	// historyCmd represents the "bss history" command
	var historyCmd = &cobra.Command{
		Use:   "history",
		Args:  cobra.NoArgs,
		Short: "Fetch the endpoint history of BSS",
		Long: `Fetch the endpoint history of BSS.

See ochami-bss(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			bssClient := bss_lib.GetClient(cmd)

			// If no ID flags are specified, get all boot parameters
			qstr := ""
			if cmd.Flag("xname").Changed || cmd.Flag("endpoint").Changed {
				values := url.Values{}
				if cmd.Flag("xname").Changed {
					x, err := cmd.Flags().GetString("xname")
					if err != nil {
						log.Logger.Error().Err(err).Msg("unable to fetch xname")
						cli.LogHelpError(cmd)
						os.Exit(1)
					}
					values.Add("name", x)
				}
				if cmd.Flag("endpoint").Changed {
					e, err := cmd.Flags().GetString("endpoint")
					if err != nil {
						log.Logger.Error().Err(err).Msg("unable to fetch endpoint")
						cli.LogHelpError(cmd)
						os.Exit(1)
					}
					values.Add("endpoint", e)
				}
				qstr = values.Encode()
			}

			// Send request
			httpEnv, err := bssClient.GetEndpointHistory(qstr)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("BSS endpoint history request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to request endpoint history from BSS")
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
	historyCmd.Flags().String("xname", "", "filter by xname")
	historyCmd.Flags().String("endpoint", "", "filter by endpoint")
	historyCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	historyCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return historyCmd
}
