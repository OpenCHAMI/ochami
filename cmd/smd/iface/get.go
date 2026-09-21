// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package iface

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	smd_lib "github.com/openchami/ochami/internal/cli/smd"
)

func newCmdIfaceGet() *cobra.Command {
	// ifaceGetCmd represents the "smd iface get" command
	var ifaceGetCmd = &cobra.Command{
		Use:     "get",
		Aliases: []string{"list"},
		Args:    cobra.NoArgs,
		Short:   "Get some or all ethernet interfaces",
		Long: `Get some or all ethernet interfaces optionally based on filter(s). If no options are
passed, all ethernet interfaces are returned. Optionally, options can be passed to limit the
ethernet interfaces returned.

See ochami-smd(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			smdClient, err := smd_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Deal with --id
			if cmd.Flag("id").Changed {
				// This endpoint requires authentication, so a token is needed
				if err := cli.SetToken(cmd); err != nil {
					return err
				}
				if err := cli.CheckToken(cmd); err != nil {
					return err
				}

				id, err := cmd.Flags().GetString("id")
				if err != nil {
					return cli.Errorf(cli.CodeUsage, "failed to get id: %w", err)
				}
				byIP := false
				if cmd.Flag("by-ip").Changed {
					byIP = true
				}
				httpEnv, err := smdClient.GetEthernetInterfaceByID(id, cli.Token, byIP)
				if err != nil {
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						return cli.Errorf(cli.CodeHTTP, "SMD ethernet interface request by ID yielded unsuccessful HTTP response: %w", err)
					}
					return cli.Errorf(cli.CodeNetwork, "failed to request ethernet interfaces by ID from SMD: %w", err)
				}
				fmt.Println(string(httpEnv.Body))
				return nil
			} else if cmd.Flag("by-ip").Changed {
				return cli.Errorf(cli.CodeUsage, "--by-ip can only be used with --id")
			}

			// All other cases
			qstr := ""
			if cmd.Flag("mac").Changed || cmd.Flag("ip").Changed || cmd.Flag("net").Changed || cmd.Flag("comp-id").Changed ||
				cmd.Flag("type").Changed || cmd.Flag("older-than").Changed || cmd.Flag("newer-than").Changed {
				values := url.Values{}
				if cmd.Flag("mac").Changed {
					s, _ := cmd.Flags().GetStringSlice("mac")
					for _, m := range s {
						values.Add("MACAddress", m)
					}
				}
				if cmd.Flag("ip").Changed {
					s, _ := cmd.Flags().GetStringSlice("ip")
					for _, i := range s {
						values.Add("IPAddress", i)
					}
				}
				if cmd.Flag("net").Changed {
					s, _ := cmd.Flags().GetStringSlice("net")
					for _, n := range s {
						values.Add("Network", n)
					}
				}
				if cmd.Flag("comp-id").Changed {
					s, _ := cmd.Flags().GetStringSlice("comp-id")
					for _, c := range s {
						values.Add("ComponentID", c)
					}
				}
				if cmd.Flag("type").Changed {
					s, _ := cmd.Flags().GetStringSlice("type")
					for _, t := range s {
						values.Add("Type", t)
					}
				}
				if cmd.Flag("older-than").Changed {
					s, _ := cmd.Flags().GetString("older-than")
					values.Add("OlderThan", s)
				}
				if cmd.Flag("newer-than").Changed {
					s, _ := cmd.Flags().GetString("newer-than")
					values.Add("NewerThan", s)
				}
				qstr = values.Encode()
			}
			httpEnv, err := smdClient.GetEthernetInterfaces(qstr)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					return cli.Errorf(cli.CodeHTTP, "SMD ethernet interface request yielded unsuccessful HTTP response: %w", err)
				}
				return cli.Errorf(cli.CodeNetwork, "failed to request ethernet interfaces from SMD: %w", err)
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
	ifaceGetCmd.Flags().StringP("id", "i", "", "get an ethernet interface by its ID")
	ifaceGetCmd.Flags().Bool("by-ip", false, "get all IP addresses for an ethernet interface (used with --id)")
	ifaceGetCmd.Flags().StringSliceP("mac", "m", []string{}, "filter ethernet interfaces by mac address")
	ifaceGetCmd.Flags().StringSlice("ip", []string{}, "filter ethernet interfaces by IP address")
	ifaceGetCmd.Flags().StringSlice("net", []string{}, "filter ethernet interfaces by IP on given network")
	ifaceGetCmd.Flags().StringSlice("comp-id", []string{}, "filter ethernet interfaces by component ID")
	ifaceGetCmd.Flags().StringSlice("type", []string{}, "filter ethernet interfaces by type")
	ifaceGetCmd.Flags().String("older-than", "", "filter ethernet interfaces by update time older than specified time (RFC3339-formatted)")
	ifaceGetCmd.Flags().String("newer-than", "", "filter ethernet interfaces by update time older than specified time (RFC3339-formatted)")
	ifaceGetCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	ifaceGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)
	ifaceGetCmd.MarkFlagsMutuallyExclusive("id", "mac")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("id", "ip")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("id", "net")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("id", "comp-id")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("id", "type")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("id", "older-than")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("id", "newer-than")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("by-ip", "mac")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("by-ip", "ip")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("by-ip", "net")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("by-ip", "comp-id")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("by-ip", "type")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("by-ip", "older-than")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("by-ip", "newer-than")

	return ifaceGetCmd
}
