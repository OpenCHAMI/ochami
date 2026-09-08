// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package hosts

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	bss_lib "github.com/openchami/ochami/internal/cli/bss"
)

func newCmdHostsGet() *cobra.Command {
	// hostsGetCmd represents the "bss hosts get" command
	var hostsGetCmd = &cobra.Command{
		Use:     "get",
		Aliases: []string{"list"},
		Args:    cobra.NoArgs,
		Short:   "Get information on hosts known to BSS",
		Long: `Get information on hosts known to BSS.

See ochami-bss(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			bssClient, err := bss_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// If no ID flags are specified, get all boot parameters
			qstr := ""
			if cmd.Flag("xname").Changed ||
				cmd.Flag("mac").Changed ||
				cmd.Flag("nid").Changed {
				values := url.Values{}
				if cmd.Flag("xname").Changed {
					x, err := cmd.Flags().GetString("xname")
					if err != nil {
						return cli.Errorf(cli.CodeUsage, "unable to fetch xname: %w", err)
					}
					values.Add("name", x)
				}
				if cmd.Flag("mac").Changed {
					m, err := cmd.Flags().GetString("mac")
					if err != nil {
						return cli.Errorf(cli.CodeUsage, "unable to fetch mac: %w", err)
					}
					values.Add("mac", m)
				}
				if cmd.Flag("nid").Changed {
					n, err := cmd.Flags().GetInt32("nid")
					if err != nil {
						return cli.Errorf(cli.CodeUsage, "unable to fetch nid: %w", err)
					}
					values.Add("nid", fmt.Sprintf("%d", n))
				}
				qstr = values.Encode()
			}
			httpEnv, err := bssClient.GetHosts(qstr)
			if err != nil {
				return cli.ClassifyClientError(err, "BSS hosts request yielded unsuccessful HTTP response", "failed to request hosts from BSS")

			}

			// Print output
			outBytes, err := client.FormatBody(httpEnv.Body, cli.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
			}
			fmt.Fprint(cli.Ios.Out(), string(outBytes))

			return nil
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
