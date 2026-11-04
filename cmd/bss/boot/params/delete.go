// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package params

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"

	bss_lib "github.com/openchami/ochami/internal/cli/bss"
)

// bootParamsDeleteOptions holds the flag values for the bss boot params delete command.
type bootParamsDeleteOptions struct {
	bootParamFields
	NoConfirm bool
}

// runCoreBootParamsDelete contains the core logic for the bss boot params delete command.
// It takes the parsed options and performs the actual work of deleting boot parameters.
func runCoreBootParamsDelete(cmd *cobra.Command, opts *bootParamsDeleteOptions, rt *cli.Runtime) error {
	// The BSS BootParams structs we will send, from the payload or, without
	// -d, from the flags. The two are mutually exclusive.
	bps, err := readBootParams(cmd, rt, opts.bootParamFields)
	if err != nil {
		return err
	}

	// Ask before attempting deletion unless --no-confirm was passed
	if !opts.NoConfirm {
		rt.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
		respDelete, err := rt.Ios.LoopYesNo(cmd.Context(), "Really delete?")
		if err != nil {
			return cli.Errorf(cli.CodeGeneric, "error fetching user input: %w", err)
		} else if !respDelete {
			return cli.Errorf(cli.CodeDeclined, "user aborted boot parameter deletion")
		} else {
			rt.Logger.Debug().Msg("User answered affirmatively to delete boot parameters")
		}
	}

	// Create client to use for requests with runtime
	bssClient, err := bss_lib.GetClient(cmd, rt)
	if err != nil {
		return err
	}

	// Handle token for this command
	if err := rt.HandleToken(cmd); err != nil {
		return err
	}

	// Send 'em off, one request per boot parameters object
	results := bssClient.DeleteBootParamsBatch(cmd.Context(), bps, rt.Token)
	return cli.AggregateItemErrors(rt.Logger, results.Errors(), "BSS boot parameter deletion")
}

func newCmdBootParamsDelete() *cobra.Command {
	// bootParamsDelete represents the "bss boot params delete" command
	var bootParamsDelete = &cobra.Command{
		Use:   "delete",
		Args:  cobra.NoArgs,
		Short: "Delete boot parameters for one or more components",
		Long: `Delete boot parameters for one or more components. At least one of --kernel,
--initrd, --params, --xname, --mac, or --nid must be specified.
This command can delete boot parameters by config (kernel URI,
initrd URI, or kernel command line) or by component (--xname,
--mac, or --nid). The user will be asked for confirmation before
deletion unless --no-confirm is passed. Alternatively, pass -d to pass
raw payload data or (if flag argument starts with @) a file containing
the payload data. -f can be specified to change the format of the
input payload data ('json' by default), but the rules above still
apply for the payload. If "-" is used as the input payload filename,
the data is read from standard input. -d cannot be used with
--xname, --mac, --nid, --kernel, --initrd, or --params.
The payload can hold one boot parameters object or an array of
them, each sent in its own request.

This command sends a DELETE to BSS. An access token is required.

See ochami-bss(1) for more details.`,
		Example: `  # Delete boot parameters using CLI flags
  ochami bss boot params delete --kernel https://example.com/kernel
  ochami bss boot params delete --kernel https://example.com/kernel --initrd https://example.com/initrd

  # Delete boot parameters using input payload data
  ochami bss boot params delete -d '{"macs":["00:de:ad:be:ef:00"]}'
  ochami bss boot params delete -d '{"kernel":"https://example.com/kernel"}'

  # Delete multiple boot parameters using JSON array
  ochami bss boot params delete -d '[
    {
      "macs": ["00:de:ad:be:ef:00"]
    },
    {
      "kernel": "https://example.com/kernel"
    }
  ]'

  # Delete boot parameters using input payload data
  ochami bss boot params delete -d @payload.json
  ochami bss boot params delete -d @payload.yaml -f yaml

  # Delete boot parameters using data from standard input
  echo '<json_data>' | ochami bss boot params delete -d @-
  echo '<yaml_data>' | ochami bss boot params delete -d @- -f yaml`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// With -d, the flag groups registered by markDataExclusive reject
			// the other flags
			if !cmd.Flag("data").Changed && !cli.AnyFlagChanged(cmd, bootParamFlags...) {
				// If -d/--data not passed, then boot parameters can be deleted
				// by component (--xname/--nid/--mac) or by config
				// (--kernel/--initrd/--params), so any one of them is enough
				return cli.Errorf(cli.CodeUsage, "expected -d or at least one of --xname, --nid, --mac, --kernel, --initrd, or --params")
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Extract options from flags
			opts := &bootParamsDeleteOptions{bootParamFields: readBootParamFlags(cmd)}
			// --no-confirm is registered as a bool on this command, so GetBool
			// can't fail and its error is ignored
			if cmd.Flag("no-confirm").Changed {
				opts.NoConfirm, _ = cmd.Flags().GetBool("no-confirm")
			}

			return runCoreBootParamsDelete(cmd, opts, rt)
		},
	}

	// Create flags
	bootParamsDelete.Flags().String("kernel", "", "URI of kernel")
	bootParamsDelete.Flags().String("initrd", "", "URI of initrd/initramfs")
	bootParamsDelete.Flags().String("params", "", "kernel parameters")
	bootParamsDelete.Flags().StringSliceP("xname", "x", []string{}, "one or more xnames whose boot parameters to delete")
	bootParamsDelete.Flags().StringSliceP("mac", "m", []string{}, "one or more MAC addresses whose boot parameters to delete")
	bootParamsDelete.Flags().Int32SliceP("nid", "n", []int32{}, "one or more node IDs whose boot parameters to delete")
	bootParamsDelete.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	cli.AddFormatInputFlag(bootParamsDelete)
	markDataExclusive(bootParamsDelete)
	bootParamsDelete.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return bootParamsDelete
}
