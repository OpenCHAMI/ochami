// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package params

import (
	"github.com/openchami/bss/pkg/bssTypes"
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"

	bss_lib "github.com/openchami/ochami/internal/cli/bss"
	"github.com/openchami/ochami/pkg/client/bss"
)

// bootParamsAddOptions holds the flag values for the bss boot params add command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type bootParamsAddOptions struct {
	Xname  []string
	Mac    []string
	Nid    []int32
	Kernel string
	Initrd string
	Params string
}

// toBootParams converts the options to a BSS BootParams struct.
func (opts *bootParamsAddOptions) toBootParams() bssTypes.BootParams {
	return bssTypes.BootParams{
		Hosts:  opts.Xname,
		Macs:   opts.Mac,
		Nids:   opts.Nid,
		Kernel: opts.Kernel,
		Initrd: opts.Initrd,
		Params: opts.Params,
	}
}

// runCoreBootParamsAdd contains the core logic for the bss boot params add command.
// It takes the parsed options and performs the actual work of adding boot parameters.
func runCoreBootParamsAdd(cmd *cobra.Command, opts *bootParamsAddOptions, bssClient *bss.BSSClient) error {
	// Handle token for this command
	if err := cli.HandleToken(cmd); err != nil {
		return err
	}

	// The BSS BootParams struct we will send
	bp := opts.toBootParams()

	// Read payload from file first, allowing overwrites from flags
	if cmd.Flag("data").Changed {
		if err := cli.HandlePayload(cmd, &bp); err != nil {
			return err
		}
	}

	// Validate MAC addresses if any were provided
	if len(opts.Mac) > 0 {
		if err := bp.CheckMacs(); err != nil {
			return cli.Errorf(cli.CodeUsage, "invalid mac(s): %w", err)
		}
	}

	// Send 'em off
	_, err := bssClient.PostBootParams(cmd.Context(), bp, cli.Token)
	if err != nil {
		return cli.ClassifyClientError(err, "BSS boot parameter request yielded unsuccessful HTTP response", "failed to add boot parameters to BSS")
	}

	return nil
}

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
		PreRunE: validateBootParamsSetFlags,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			bssClient, err := bss_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Extract options from flags
			// Since flags are registered with the correct types on this command,
			// these Get* calls cannot fail, so we ignore errors with explicit comments
			opts := &bootParamsAddOptions{}
			if cmd.Flag("xname").Changed {
				opts.Xname, _ = cmd.Flags().GetStringSlice("xname") // Flag registered with matching type, error impossible
			}
			if cmd.Flag("mac").Changed {
				opts.Mac, _ = cmd.Flags().GetStringSlice("mac") // Flag registered with matching type, error impossible
			}
			if cmd.Flag("nid").Changed {
				opts.Nid, _ = cmd.Flags().GetInt32Slice("nid") // Flag registered with matching type, error impossible
			}
			if cmd.Flag("kernel").Changed {
				opts.Kernel, _ = cmd.Flags().GetString("kernel") // Flag registered with matching type, error impossible
			}
			if cmd.Flag("initrd").Changed {
				opts.Initrd, _ = cmd.Flags().GetString("initrd") // Flag registered with matching type, error impossible
			}
			if cmd.Flag("params").Changed {
				opts.Params, _ = cmd.Flags().GetString("params") // Flag registered with matching type, error impossible
			}

			return runCoreBootParamsAdd(cmd, opts, bssClient)
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
