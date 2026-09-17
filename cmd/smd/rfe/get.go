// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rfe

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	smd_lib "github.com/openchami/ochami/internal/cli/smd"
)

func newCmdRfeGet() *cobra.Command {
	// rfeGetCmd represents the "smd rfe get" command
	var rfeGetCmd = &cobra.Command{
		Use:     "get",
		Aliases: []string{"list"},
		Args:    cobra.NoArgs,
		Short:   "Get all redfish endpoints or some based on filter(s)",
		Long: `Get all redfish endpoints or some based on filter(s). If no options are passed,
all redfish endpoints are returned. Optionally, options can be passed to limit the redfish
endpoints returned.

See ochami-smd(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			smdClient, err := smd_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// If no ID flags are specified, get all redfish endpoints
			qstr := ""
			if cmd.Flag("xname").Changed || cmd.Flag("mac").Changed || cmd.Flag("ip").Changed ||
				cmd.Flag("fqdn").Changed || cmd.Flag("type").Changed || cmd.Flag("uuid").Changed {
				values := url.Values{}
				if cmd.Flag("xname").Changed {
					s, _ := cmd.Flags().GetStringSlice("xname")
					for _, x := range s {
						values.Add("id", x)
					}
				}
				if cmd.Flag("mac").Changed {
					s, _ := cmd.Flags().GetStringSlice("mac")
					for _, m := range s {
						values.Add("macaddr", m)
					}
				}
				if cmd.Flag("ip").Changed {
					s, _ := cmd.Flags().GetStringSlice("ip")
					for _, i := range s {
						values.Add("ipaddress", i)
					}
				}
				if cmd.Flag("fqdn").Changed {
					s, _ := cmd.Flags().GetStringSlice("fqdn")
					for _, f := range s {
						values.Add("fqdn", f)
					}
				}
				if cmd.Flag("type").Changed {
					s, _ := cmd.Flags().GetStringSlice("type")
					for _, t := range s {
						values.Add("type", t)
					}
				}
				if cmd.Flag("uuid").Changed {
					s, _ := cmd.Flags().GetStringSlice("uuid")
					for _, u := range s {
						values.Add("uuid", u)
					}
				}
				qstr = values.Encode()
			}
			httpEnv, err := smdClient.GetRedfishEndpoints(qstr, cli.Token)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					return cli.Errorf(cli.CodeHTTP, "SMD redfish endpoint request yielded unsuccessful HTTP response: %w", err)
				}
				return cli.Errorf(cli.CodeNetwork, "failed to request redfish endpoints from SMD: %w", err)
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
	rfeGetCmd.Flags().StringSliceP("xname", "x", []string{}, "filter redfish endpoints by xname")
	rfeGetCmd.Flags().StringSlice("fqdn", []string{}, "filter redfish endpoints by fully-qualified domain name")
	rfeGetCmd.Flags().StringSlice("type", []string{}, "filter redfish endpoints by type (e.b. Node, NodeBMC, etc.)")
	rfeGetCmd.Flags().StringSlice("uuid", []string{}, "filter redfish endpoints by UUID")
	rfeGetCmd.Flags().StringSliceP("mac", "m", []string{}, "filter redfish endpoints by MAC address")
	rfeGetCmd.Flags().StringSliceP("ip", "i", []string{}, "filter redfish endpoints by IP address")
	rfeGetCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	rfeGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return rfeGetCmd
}
