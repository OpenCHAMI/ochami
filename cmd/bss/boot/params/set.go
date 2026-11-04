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

// bootParamsSetOptions holds the flag values for the bss boot params set command.
type bootParamsSetOptions struct {
	bootParamFields
	Data string
}

// runCoreBootParamsSet contains the core logic for the bss boot params set command.
// It takes the parsed options and performs the actual work of setting boot parameters.
func runCoreBootParamsSet(cmd *cobra.Command, opts *bootParamsSetOptions, bssClient *bss.BSSClient, rt *cli.Runtime) error {
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
	results := bssClient.PutBootParamsBatch(cmd.Context(), bps, rt.Token)
	return cli.AggregateItemErrors(rt.Logger, results.Errors(), "BSS boot parameter setting")
}

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
the data is read from standard input. -d cannot be used with
--xname, --mac, --nid, --kernel, --initrd, or --params.
The payload can hold one boot parameters object or an array of
them, each sent in its own request.

This command sends a PUT to BSS. An access token is required.

See ochami-bss(1) for more details.`,
		Example: `  # Set boot params using CLI flags
  ochami bss boot params set --xname x1000c1s7b0 --kernel https://example.com/kernel
  ochami bss boot params set --xname x1000c1s7b0,x1000c1s7b1 --kernel https://example.com/kernel
  ochami bss boot params set --xname x1000c1s7b0 --xname x1000c1s7b1 --kernel https://example.com/kernel
  ochami bss boot params set --xname x1000c1s7b0 --nid 1 --mac 00:c0:ff:ee:00:00 --params 'quiet nosplash'

  # Set boot parameters using input payload data
  ochami bss boot params set -d '{
    "macs": ["00:de:ad:be:ef:00"],
    "kernel": "https://example.com/kernel"
  }'

  # Set multiple boot parameters using JSON array
  ochami bss boot params set -d '[
    {
      "macs": ["00:de:ad:be:ef:00"],
      "kernel": "https://example.com/kernel"
    },
    {
      "macs": ["00:c0:ff:ee:00:00"],
      "kernel": "https://example.com/kernel2"
    }
  ]'

  # Set boot params using input payload file
  ochami bss boot params set -d @payload.json
  ochami bss boot params set -d @payload.yaml -f yaml

  # Set boot params using data from standard input
  echo <json_data> | ochami bss boot params set -d @-
  echo <yaml_data> | ochami bss boot params set -d @- -f yaml`,
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
			opts := &bootParamsSetOptions{bootParamFields: readBootParamFlags(cmd)}

			return runCoreBootParamsSet(cmd, opts, bssClient, rt)
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

	cli.AddFormatInputFlag(bootParamsSetCmd)
	markDataExclusive(bootParamsSetCmd)
	bootParamsSetCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return bootParamsSetCmd
}
