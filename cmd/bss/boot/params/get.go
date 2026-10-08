// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package params

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	bss_lib "github.com/openchami/ochami/internal/cli/bss"
)

func newCmdBootParamsGet() *cobra.Command {
	// bootParamsGetCmd represents the "bss boot params get" command
	var bootParamsGetCmd = &cobra.Command{
		Use:     "get",
		Aliases: []string{"list"},
		Args:    cobra.NoArgs,
		Short:   "Get boot parameters for one or all nodes",
		Long: `Get boot parameters for one or all nodes. If no options are passed, all boot
parameters are returned. Optionally, --mac, --xname, and/or --nid can be passed at least once
to get boot parameters for specific components.

This command sends a GET to BSS. An access token is required.

See ochami-bss(1) for more details.`,
		Example: `  ochami bss boot params get
  ochami bss boot params get --mac 00:de:ad:be:ef:00
  ochami bss boot params get --mac 00:de:ad:be:ef:00,00:c0:ff:ee:00:00
  ochami bss boot params get --mac 00:de:ad:be:ef:00 --mac 00:c0:ff:ee:00:00`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			bssClient, err := bss_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// If no ID flags are specified, get all boot parameters
			qstr := ""
			if cmd.Flag("xname").Changed ||
				cmd.Flag("mac").Changed ||
				cmd.Flag("nid").Changed {
				values := url.Values{}
				if cmd.Flag("xname").Changed {
					s, err := cmd.Flags().GetStringSlice("xname")
					if err != nil {
						return cli.Errorf(cli.CodeUsage, "unable to fetch xname list: %w", err)
					}
					for _, x := range s {
						values.Add("name", x)
					}
				}
				if cmd.Flag("mac").Changed {
					s, err := cmd.Flags().GetStringSlice("mac")
					if err != nil {
						return cli.Errorf(cli.CodeUsage, "unable to fetch mac list: %w", err)
					}
					for _, m := range s {
						values.Add("mac", m)
					}
				}
				if cmd.Flag("nid").Changed {
					s, err := cmd.Flags().GetInt32Slice("nid")
					if err != nil {
						return cli.Errorf(cli.CodeUsage, "unable to fetch nid list: %w", err)
					}
					for _, n := range s {
						values.Add("nid", fmt.Sprintf("%d", n))
					}
				}
				qstr = values.Encode()
			}
			httpEnv, err := bssClient.GetBootParams(qstr, cli.Token)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					return cli.Errorf(cli.CodeHTTP, "BSS boot parameter request yielded unsuccessful HTTP response: %w", err)
				}
				return cli.Errorf(cli.CodeNetwork, "failed to request boot parameters from BSS: %w", err)
			}

			// Print output
			outBytes, err := client.FormatBody(httpEnv.Body, cli.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
			}
			fmt.Print(string(outBytes))

			return nil
		},
	}

	// Create flags
	bootParamsGetCmd.Flags().StringSliceP("xname", "x", []string{}, "one or more xnames whose boot parameters to get")
	bootParamsGetCmd.Flags().StringSliceP("mac", "m", []string{}, "one or more MAC addresses whose boot parameters to get")
	bootParamsGetCmd.Flags().Int32SliceP("nid", "n", []int32{}, "one or more node IDs whose boot parameters to get")
	bootParamsGetCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	bootParamsGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return bootParamsGetCmd
}
