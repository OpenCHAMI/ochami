// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rfe

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	smd_lib "github.com/openchami/ochami/internal/cli/smd"
	"github.com/openchami/ochami/pkg/client/smd"
)

// rfeGetOptions holds the flag values for the smd rfe get command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type rfeGetOptions struct {
	Xname []string
	Mac   []string
	IP    []string
	FQDN  []string
	Type  []string
	UUID  []string
}

// runCoreRfeGet contains the core logic for the smd rfe get command.
// It takes the parsed options and performs the actual work of getting redfish endpoints.
func runCoreRfeGet(cmd *cobra.Command, opts *rfeGetOptions, smdClient *smd.SMDClient) error {
	// Handle token for this command
	if err := cli.HandleToken(cmd); err != nil {
		return err
	}

	// If no ID flags are specified, get all redfish endpoints
	qstr := ""
	if len(opts.Xname) > 0 || len(opts.Mac) > 0 || len(opts.IP) > 0 ||
		len(opts.FQDN) > 0 || len(opts.Type) > 0 || len(opts.UUID) > 0 {
		values := url.Values{}
		for _, x := range opts.Xname {
			values.Add("id", x)
		}
		for _, m := range opts.Mac {
			values.Add("macaddr", m)
		}
		for _, i := range opts.IP {
			values.Add("ipaddress", i)
		}
		for _, f := range opts.FQDN {
			values.Add("fqdn", f)
		}
		for _, t := range opts.Type {
			values.Add("type", t)
		}
		for _, u := range opts.UUID {
			values.Add("uuid", u)
		}
		qstr = values.Encode()
	}

	httpEnv, err := smdClient.GetRedfishEndpoints(cmd.Context(), qstr, cli.Token)
	if err != nil {
		return cli.ClassifyClientError(err, "SMD redfish endpoint request yielded unsuccessful HTTP response", "failed to request redfish endpoints from SMD")
	}

	// Print output
	outBytes, err := client.FormatBody(httpEnv.Body, cli.FormatOutput)
	if err != nil {
		return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
	}
	fmt.Fprint(cli.Ios.Out(), string(outBytes))

	return nil
}

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

			// Extract options from flags
			// Since flags are registered with the correct types on this command,
			// these Get* calls cannot fail, so we ignore errors with explicit comments
			opts := &rfeGetOptions{}
			if cmd.Flag("xname").Changed {
				opts.Xname, _ = cmd.Flags().GetStringSlice("xname") // Flag registered with matching type, error impossible
			}
			if cmd.Flag("mac").Changed {
				opts.Mac, _ = cmd.Flags().GetStringSlice("mac") // Flag registered with matching type, error impossible
			}
			if cmd.Flag("ip").Changed {
				opts.IP, _ = cmd.Flags().GetStringSlice("ip") // Flag registered with matching type, error impossible
			}
			if cmd.Flag("fqdn").Changed {
				opts.FQDN, _ = cmd.Flags().GetStringSlice("fqdn") // Flag registered with matching type, error impossible
			}
			if cmd.Flag("type").Changed {
				opts.Type, _ = cmd.Flags().GetStringSlice("type") // Flag registered with matching type, error impossible
			}
			if cmd.Flag("uuid").Changed {
				opts.UUID, _ = cmd.Flags().GetStringSlice("uuid") // Flag registered with matching type, error impossible
			}

			return runCoreRfeGet(cmd, opts, smdClient)
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
