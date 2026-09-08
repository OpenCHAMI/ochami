// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package params

import (
	"github.com/openchami/bss/pkg/bssTypes"
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"

	bss_lib "github.com/openchami/ochami/internal/cli/bss"
)

func newCmdBootParamsAdd() *cobra.Command {
	// bootParamsAddCmd represents the "bss boot params add" command
	var bootParamsAddCmd = &cobra.Command{
		Use:   "add",
		Args:  cobra.NoArgs,
		Short: "Add new boot parameters for one or more components",
		Long: `Add new boot parameters for one or more components. At least one of --kernel,
--initrd, or --params must be specified as well as at least one of --xname,
--mac, or --nid. Alternatively, pass -d to pass raw payload data or (if
flag argument starts with @) a file containing the payload data. -f can
be specified to change the format of the input payload data ('json' by
default), but the rules above still apply for the payload. If "-" is used
as the input payload filename, the data is read from standard input.

This command sends a POST to BSS. An access token is required.

See ochami-bss(1) for more details.`,
		Example: `  # Add boot parameters using CLI flags
  ochami bss boot params add \
    --mac 00:de:ad:be:ef:00 \
    --kernel https://example.com/kernel \
    --initrd https://example.com/initrd \
    --params 'quiet nosplash'
  ochami bss boot params add --mac 00:de:ad:be:ef:00,00:c0:ff:ee:00:00 --params 'quiet nosplash'
  ochami bss boot params add --mac 00:de:ad:be:ef:00 --mac 00:c0:ff:ee:00:00 --kernel https://example.com/kernel

  # Add boot parameters using input payload data
  ochami bss boot params add -d '{"macs":["00:de:ad:be:ef:00"],"kernel":"https://example.com/kernel"}'

  # Add boot parameters using input payload file
  ochami bss boot params add -d @payload.json
  ochami bss boot params add -d @payload.yaml -f yaml

  # Add boot parameters using data from standard input
  echo '<json_data>' | ochami bss boot params add -d @-
  echo '<yaml_data>' | ochami bss boot params add -d @- -f yaml`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Function to return true if any flag is set
			anyChanged := func(flags ...string) bool {
				for _, f := range flags {
					if cmd.Flag(f).Changed {
						return true
					}
				}
				return false
			}
			if cmd.Flag("data").Changed {
				// -d/--data trumps all, ignore values of other flags if specified
				if anyChanged("xname", "nid", "mac", "kernel", "initrd", "params") {
					log.Logger.Warn().Msgf("raw data passed, ignoring CLI configuration")
				}
			} else {
				// If -d/--data not passed, then at least one of --xname/--nid/--mac must
				// be specified, along with at least one of --kernel/--initrd/--params
				if !anyChanged("xname", "nid", "mac") {
					return cli.Errorf(cli.CodeUsage, "expected -d or one of --xname, --nid, or --mac")
				} else if !anyChanged("kernel", "initrd", "params") {
					return cli.Errorf(cli.CodeUsage, "specifying any of --xname, --nid, or --mac also requires specifying at least one of --kernel, --initrd, or --params")
				}
			}

			return nil
		},
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

			// The BSS BootParams struct we will send
			bp := bssTypes.BootParams{}

			// Read payload from file first, allowing overwrites from flags
			if cmd.Flag("data").Changed {
				if err := cli.HandlePayload(cmd, &bp); err != nil {
					return err
				}
			}

			// Set the hosts the boot parameters are for
			if cmd.Flag("xname").Changed {
				bp.Hosts, _ = cmd.Flags().GetStringSlice("xname") //nolint:errcheck // flag is registered with the matching type on this command
			}
			if cmd.Flag("mac").Changed {
				bp.Macs, _ = cmd.Flags().GetStringSlice("mac") //nolint:errcheck // flag is registered with the matching type on this command
				if err = bp.CheckMacs(); err != nil {
					return cli.Errorf(cli.CodeUsage, "invalid mac(s): %w", err)
				}
			}
			if cmd.Flag("nid").Changed {
				bp.Nids, _ = cmd.Flags().GetInt32Slice("nid") //nolint:errcheck // flag is registered with the matching type on this command
			}

			// Set the boot parameters
			if cmd.Flag("kernel").Changed {
				bp.Kernel, _ = cmd.Flags().GetString("kernel") //nolint:errcheck // flag is registered with the matching type on this command
			}
			if cmd.Flag("initrd").Changed {
				bp.Initrd, _ = cmd.Flags().GetString("initrd") //nolint:errcheck // flag is registered with the matching type on this command
			}
			if cmd.Flag("params").Changed {
				bp.Params, _ = cmd.Flags().GetString("params") //nolint:errcheck // flag is registered with the matching type on this command
			}

			// Send 'em off
			_, err = bssClient.PostBootParams(bp, cli.Token)
			if err != nil {
				return cli.ClassifyClientError(err, "BSS boot parameter request yielded unsuccessful HTTP response", "failed to add boot parameters to BSS")

			}

			return nil
		},
	}

	// Create flags
	bootParamsAddCmd.Flags().String("kernel", "", "URI of kernel")
	bootParamsAddCmd.Flags().String("initrd", "", "URI of initrd/initramfs")
	bootParamsAddCmd.Flags().String("params", "", "kernel parameters")
	bootParamsAddCmd.Flags().StringSliceP("xname", "x", []string{}, "one or more xnames whose boot parameters to add")
	bootParamsAddCmd.Flags().StringSliceP("mac", "m", []string{}, "one or more MAC addresses whose boot parameters to add")
	bootParamsAddCmd.Flags().Int32SliceP("nid", "n", []int32{}, "one or more node IDs whose boot parameters to add")
	bootParamsAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	bootParamsAddCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	bootParamsAddCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return bootParamsAddCmd
}
