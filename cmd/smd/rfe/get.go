// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rfe

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"

	smd_lib "github.com/OpenCHAMI/ochami/internal/cli/smd"
)

func newCmdRfeGet() *cobra.Command {
	// rfeGetCmd represents the "smd rfe get" command
	var rfeGetCmd = &cobra.Command{
		Use:   "get",
		Args:  cobra.NoArgs,
		Short: "Get all redfish endpoints or some based on filter(s)",
		Long: `Get all redfish endpoints or some based on filter(s). If no options are passed,
all redfish endpoints are returned. Optionally, options can be passed to limit the redfish
endpoints returned.

See ochami-smd(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			smdClient := smd_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// If no ID flags are specified, get all redfish endpoints
			qstr := ""
			if cmd.Flag("xname").Changed || cmd.Flag("mac").Changed || cmd.Flag("ip").Changed ||
				cmd.Flag("fqdn").Changed || cmd.Flag("type").Changed || cmd.Flag("uuid").Changed {
				values := url.Values{}
				if cmd.Flag("xname").Changed {
					s, err := cmd.Flags().GetStringSlice("xname")
					if err != nil {
						log.Logger.Error().Err(err).Msg("unable to fetch xname list")
						cli.LogHelpError(cmd)
						os.Exit(1)
					}
					for _, x := range s {
						values.Add("id", x)
					}
				}
				if cmd.Flag("mac").Changed {
					s, err := cmd.Flags().GetStringSlice("mac")
					if err != nil {
						log.Logger.Error().Err(err).Msg("unable to fetch mac list")
						cli.LogHelpError(cmd)
						os.Exit(1)
					}
					for _, m := range s {
						values.Add("macaddr", m)
					}
				}
				if cmd.Flag("ip").Changed {
					s, err := cmd.Flags().GetStringSlice("ip")
					if err != nil {
						log.Logger.Error().Err(err).Msg("unable to fetch ip list")
						cli.LogHelpError(cmd)
						os.Exit(1)
					}
					for _, i := range s {
						values.Add("ipaddress", i)
					}
				}
				if cmd.Flag("fqdn").Changed {
					s, err := cmd.Flags().GetStringSlice("fqdn")
					if err != nil {
						log.Logger.Error().Err(err).Msg("unable to fetch fqdn list")
						cli.LogHelpError(cmd)
						os.Exit(1)
					}
					for _, f := range s {
						values.Add("fqdn", f)
					}
				}
				if cmd.Flag("type").Changed {
					s, err := cmd.Flags().GetStringSlice("type")
					if err != nil {
						log.Logger.Error().Err(err).Msg("unable to fetch type list")
						cli.LogHelpError(cmd)
						os.Exit(1)
					}
					for _, t := range s {
						values.Add("type", t)
					}
				}
				if cmd.Flag("uuid").Changed {
					s, err := cmd.Flags().GetStringSlice("uuid")
					if err != nil {
						log.Logger.Error().Err(err).Msg("unable to fetch uuid list")
						cli.LogHelpError(cmd)
						os.Exit(1)
					}
					for _, u := range s {
						values.Add("uuid", u)
					}
				}
				qstr = values.Encode()
			}
			httpEnv, err := smdClient.GetRedfishEndpoints(qstr, cli.Token)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("SMD redfish endpoint request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to request redfish endpoints from SMD")
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
