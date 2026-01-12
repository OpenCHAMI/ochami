// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package hosts

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

func newCmdHostsGet() *cobra.Command {
	// hostsGetCmd represents the "bss hosts get" command
	var hostsGetCmd = &cobra.Command{
		Use:   "get",
		Args:  cobra.NoArgs,
		Short: "Get information on hosts known to BSS",
		Long: `Get information on hosts known to BSS.

See ochami-bss(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			bssClient := bss_lib.GetClient(cmd)

			// If no ID flags are specified, get all boot parameters
			qstr := ""
			if cmd.Flag("xname").Changed ||
				cmd.Flag("mac").Changed ||
				cmd.Flag("nid").Changed {
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
				if cmd.Flag("mac").Changed {
					m, err := cmd.Flags().GetString("mac")
					if err != nil {
						log.Logger.Error().Err(err).Msg("unable to fetch mac")
						cli.LogHelpError(cmd)
						os.Exit(1)
					}
					values.Add("mac", m)
				}
				if cmd.Flag("nid").Changed {
					n, err := cmd.Flags().GetInt32("nid")
					if err != nil {
						log.Logger.Error().Err(err).Msg("unable to fetch nid")
						cli.LogHelpError(cmd)
						os.Exit(1)
					}
					values.Add("nid", fmt.Sprintf("%d", n))
				}
				qstr = values.Encode()
			}
			httpEnv, err := bssClient.GetHosts(qstr)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("BSS hosts request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to request hosts from BSS")
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
	hostsGetCmd.Flags().StringP("xname", "x", "", "xname whose host information to get")
	hostsGetCmd.Flags().StringP("mac", "m", "", "MAC address whose boot parameters to get")
	hostsGetCmd.Flags().Int32P("nid", "n", 0, "node ID whose host information to get")
	hostsGetCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	hostsGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return hostsGetCmd
}
