// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package params

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"

	bss_lib "github.com/openchami/ochami/internal/cli/bss"
	"github.com/openchami/ochami/pkg/client/bss"
)

// bootParamsAddOptions holds the flag values for the bss boot params add command.
type bootParamsAddOptions struct {
	bootParamFields
}

// runCoreBootParamsAdd contains the core logic for the bss boot params add command.
// It takes the parsed options and performs the actual work of adding boot parameters.
func runCoreBootParamsAdd(cmd *cobra.Command, opts *bootParamsAddOptions, bssClient *bss.BSSClient, rt *cli.Runtime) error {
	// Handle token for this command
	if err := rt.HandleToken(cmd); err != nil {
		return err
	}

	// The BSS BootParams structs we will send, from the payload or, without
	// -d, from the flags. The two are mutually exclusive.
	bps, err := readBootParams(cmd, rt, opts.bootParamFields)
	if err != nil {
		return err
	}

	// Send 'em off, one request per boot parameters object
	results := bssClient.PostBootParamsBatch(cmd.Context(), bps, rt.Token)
	return cli.AggregateItemErrors(rt.Logger, results.Errors(), "BSS boot parameter addition")
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
-d cannot be used with --xname, --mac, --nid, --kernel, --initrd, or --params.
The payload can hold one boot parameters object or an array of them, each
sent in its own request.

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
  ochami bss boot params add -d '{
    "macs": ["00:de:ad:be:ef:00"],
    "kernel": "https://example.com/kernel"
  }'

  # Add multiple boot parameters using JSON array
  ochami bss boot params add -d '[
    {
      "macs": ["00:de:ad:be:ef:00"],
      "kernel": "https://example.com/kernel"
    },
    {
      "macs": ["00:c0:ff:ee:00:00"],
      "kernel": "https://example.com/kernel2"
    }
  ]'

  # Add boot parameters using input payload file
  ochami bss boot params add -d @payload.json
  ochami bss boot params add -d @payload.yaml -f yaml

  # Add boot parameters using data from standard input
  echo '<json_data>' | ochami bss boot params add -d @-
  echo '<yaml_data>' | ochami bss boot params add -d @- -f yaml`,
		PreRunE: validateBootParamFlags,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Create client to use for requests with runtime
			bssClient, err := bss_lib.GetClient(cmd, rt)
			if err != nil {
				return err
			}

			// Extract options from flags
			opts := &bootParamsAddOptions{bootParamFields: readBootParamFlags(cmd)}

			return runCoreBootParamsAdd(cmd, opts, bssClient, rt)
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

	cli.AddFormatInputFlag(bootParamsAddCmd)
	markDataExclusive(bootParamsAddCmd)
	bootParamsAddCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return bootParamsAddCmd
}
