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

// bootParamsUpdateOptions holds the flag values for the bss boot params update command.
type bootParamsUpdateOptions struct {
	bootParamFields
}

// runCoreBootParamsUpdate contains the core logic for the bss boot params update command.
// It takes the parsed options and performs the actual work of updating boot parameters.
func runCoreBootParamsUpdate(cmd *cobra.Command, opts *bootParamsUpdateOptions, bssClient *bss.BSSClient, rt *cli.Runtime) error {
	// Handle token for this command
	if err := rt.HandleToken(cmd); err != nil {
		return err
	}

	// The BSS BootParams struct we will send
	var bp bssTypes.BootParams

	// Build the request from the payload or, without -d, from the flags; the
	// two are mutually exclusive
	if cmd.Flag("data").Changed {
		if err := rt.HandlePayload(cmd, &bp); err != nil {
			return err
		}
	} else {
		applyBootParamFlags(cmd, &bp, opts.bootParamFields)
	}

	// Validate MAC addresses, whether they came from --mac or the payload.
	// CheckMacs accepts an empty list.
	if err := bp.CheckMacs(); err != nil {
		return cli.Errorf(cli.CodeUsage, "invalid mac(s): %w", err)
	}

	// Send 'em off
	_, err := bssClient.PatchBootParams(cmd.Context(), bp, rt.Token)
	if err != nil {
		return cli.ClassifyClientError(err, "BSS boot parameter request yielded unsuccessful HTTP response", "failed to update boot parameters in BSS")
	}

	return nil
}

func newCmdBootParamsUpdate() *cobra.Command {
	// bootParamsUpdateCmd represents the "bss boot params update" command
	var bootParamsUpdateCmd = &cobra.Command{
		Use:   "update",
		Args:  cobra.NoArgs,
		Short: "Update some or all boot parameters for one or more components",
		Long: `Update some or all boot parameters for one or more components. At least one of
--kernel, initrd, or --params must be specified as well as at least
one of --xname, --mac, or --nid. Alternatively, pass -d to pass raw
payload data or (if flag argument starts with @) a file containing
the payload data. -f can be specified to change the format of the
input payload data ('json' by default), but the rules above still
apply for the payload. If "-" is used as the input payload filename,
the data is read from standard input. -d cannot be used with
--xname, --mac, --nid, --kernel, --initrd, or --params.

This command sends a PATCH to BSS. An access token is required.

See ochami-bss(1) for details.`,
		Example: `  # Update boot parameters using CLI flags
  ochami bss boot params update --xname x1000c1s7b0 --kernel https://example.com/kernel
  ochami bss boot params update --xname x1000c1s7b0,x1000c1s7b1 --kernel https://example.com/kernel
  ochami bss boot params update --xname x1000c1s7b0 --xname x1000c1s7b1 --kernel https://example.com/kernel
  ochami bss boot params update --xname x1000c1s7b0 --nid 1 --mac 00:c0:ff:ee:00:00 --params 'quiet nosplash'

  # Update boot parameters using input payload data
  ochami bss boot params update -d '{"macs":["00:de:ad:be:ef:00"],"kernel":"https://example.com/kernel"}'

  # Update boot parameters using input payload file
  ochami bss boot params update -d @payload.json
  ochami bss boot params update -d @payload.yaml -f yaml

  # Update boot parameters using data from standard input
  echo '<json_data>' | ochami bss boot params update -d @-
  echo '<yaml_data>' | ochami bss boot params update -d @- -f yaml`,
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
			opts := &bootParamsUpdateOptions{bootParamFields: readBootParamFlags(cmd)}

			return runCoreBootParamsUpdate(cmd, opts, bssClient, rt)
		},
	}

	// Create flags
	bootParamsUpdateCmd.Flags().String("kernel", "", "URI of kernel")
	bootParamsUpdateCmd.Flags().String("initrd", "", "URI of initrd/initramfs")
	bootParamsUpdateCmd.Flags().String("params", "", "kernel parameters")
	bootParamsUpdateCmd.Flags().StringSliceP("xname", "x", []string{}, "one or more xnames whose boot parameters to update")
	bootParamsUpdateCmd.Flags().StringSliceP("mac", "m", []string{}, "one or more MAC addresses whose boot parameters to update")
	bootParamsUpdateCmd.Flags().Int32SliceP("nid", "n", []int32{}, "one or more node IDs whose boot parameters to update")
	bootParamsUpdateCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")

	cli.AddFormatInputFlag(bootParamsUpdateCmd)
	markDataExclusive(bootParamsUpdateCmd)
	bootParamsUpdateCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return bootParamsUpdateCmd
}
