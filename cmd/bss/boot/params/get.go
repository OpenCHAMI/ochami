// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package params

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	bss_lib "github.com/openchami/ochami/internal/cli/bss"
	"github.com/openchami/ochami/pkg/client/bss"
)

// bootParamsGetOptions holds the flag values for the bss boot params get command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type bootParamsGetOptions struct {
	Xname []string
	Mac   []string
	Nid   []int32
}

// runCoreBootParamsGet contains the core logic for the bss boot params get command.
// It takes the parsed options and performs the actual work of getting boot parameters.
func runCoreBootParamsGet(cmd *cobra.Command, opts *bootParamsGetOptions, bssClient *bss.BSSClient) error {
	// Handle token for this command
	if err := cli.HandleToken(cmd); err != nil {
		return err
	}

	// If no ID flags are specified, get all boot parameters
	qstr := ""
	if len(opts.Xname) > 0 || len(opts.Mac) > 0 || len(opts.Nid) > 0 {
		values := url.Values{}
		for _, x := range opts.Xname {
			values.Add("name", x)
		}
		for _, m := range opts.Mac {
			values.Add("mac", m)
		}
		for _, n := range opts.Nid {
			values.Add("nid", fmt.Sprintf("%d", n))
		}
		qstr = values.Encode()
	}

	httpEnv, err := bssClient.GetBootParams(cmd.Context(), qstr, cli.Token)
	if err != nil {
		return cli.ClassifyClientError(err, "BSS boot parameter request yielded unsuccessful HTTP response", "failed to request boot parameters from BSS")
	}

	// Print output
	outBytes, err := client.FormatBody(httpEnv.Body, cli.FormatOutput)
	if err != nil {
		return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
	}
	fmt.Fprint(cli.Ios.Out(), string(outBytes))

	return nil
}

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

			// Extract options from flags
			// Since flags are registered with the correct types on this command,
			// these Get* calls cannot fail, so we ignore errors with explicit comments
			opts := &bootParamsGetOptions{}
			if cmd.Flag("xname").Changed {
				opts.Xname, _ = cmd.Flags().GetStringSlice("xname") // Flag registered with matching type, error impossible
			}
			if cmd.Flag("mac").Changed {
				opts.Mac, _ = cmd.Flags().GetStringSlice("mac") // Flag registered with matching type, error impossible
			}
			if cmd.Flag("nid").Changed {
				opts.Nid, _ = cmd.Flags().GetInt32Slice("nid") // Flag registered with matching type, error impossible
			}

			return runCoreBootParamsGet(cmd, opts, bssClient)
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
