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

func newCmdBootParamsSet() *cobra.Command {
	// bootParamsSetCmd represents the "bss boot params set" command
	var bootParamsSetCmd = &cobra.Command{
		Use:   "set",
		Args:  cobra.NoArgs,
		Short: "Set boot parameters for one or more components, overwriting any previous",
		Long: `Set boot parameters for one or mote components, overwriting any previously-set
parameters. At least one of --kernel, --initrd, or --params is
required to tell ochami which boot data to set. Also, at least
one of --xname, --mac, or --nid is required to tell ochami which
components need modification. Alternatively, pass -d to pass raw
payload data or (if flag argument starts with @) a file containing
the payload data. -f can be specified to change the format of the
input payload data ('json' by default), but the rules above still
apply for the payload. If "-" is used as the input payload filename,
the data is read from standard input.

This command sends a PUT to BSS. An access token is required.

See ochami-bss(1) for more details.`,
		Example: `  # Set boot params using CLI flags
  ochami bss boot params set --xname x1000c1s7b0 --kernel https://example.com/kernel
  ochami bss boot params set --xname x1000c1s7b0,x1000c1s7b1 --kernel https://example.com/kernel
  ochami bss boot params set --xname x1000c1s7b0 --xname x1000c1s7b1 --kernel https://example.com/kernel
  ochami bss boot params set --xname x1000c1s7b0 --nid 1 --mac 00:c0:ff:ee:00:00 --params 'quiet nosplash'

  # Set boot parameters using input payload data
  ochami bss boot params set -d '{"macs":["00:de:ad:be:ef:00"],"kernel":"https://example.com/kernel"}'

  # Set boot params using input payload file
  ochami bss boot params set -d @payload.json
  ochami bss boot params set -d @payload.yaml -f yaml

  # Set boot params using data from standard input
  echo <json_data> | ochami bss boot params set -d @-
  echo <yaml_data> | ochami bss boot params set -d @- -f yaml`,
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
			_, err = bssClient.PutBootParams(bp, cli.Token)
			if err != nil {
				return cli.ClassifyClientError(err, "BSS boot parameter request yielded unsuccessful HTTP response", "failed to set boot parameters in BSS")

			}

			return nil
		},
	}

	// Create flags
	bootParamsSetCmd.Flags().String("kernel", "", "URI of kernel")
	bootParamsSetCmd.Flags().String("initrd", "", "URI of initrd/initramfs")
	bootParamsSetCmd.Flags().String("params", "", "kernel parameters")
	bootParamsSetCmd.Flags().StringSliceP("xname", "x", []string{}, "one or more xnames whose boot parameters to set")
	bootParamsSetCmd.Flags().StringSliceP("mac", "m", []string{}, "one or more MAC addresses whose boot parameters to set")
	bootParamsSetCmd.Flags().Int32SliceP("nid", "n", []int32{}, "one or more node IDs whose boot parameters to set")
	bootParamsSetCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	bootParamsSetCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	bootParamsSetCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return bootParamsSetCmd
}
