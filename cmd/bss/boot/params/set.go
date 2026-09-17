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
	"github.com/openchami/ochami/pkg/client/bss"
)

// bootParamsSetOptions holds the flag values for the bss boot params set command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type bootParamsSetOptions struct {
	Xname  []string
	Mac    []string
	Nid    []int32
	Kernel string
	Initrd string
	Params string
	Data   string
}

// toBootParams converts the options to a BSS BootParams struct.
func (opts *bootParamsSetOptions) toBootParams() bssTypes.BootParams {
	return bssTypes.BootParams{
		Hosts:  opts.Xname,
		Macs:   opts.Mac,
		Nids:   opts.Nid,
		Kernel: opts.Kernel,
		Initrd: opts.Initrd,
		Params: opts.Params,
	}
}

// runCoreBootParamsSetWithRuntime contains the core logic for the bss boot params set command.
// It takes the parsed options and performs the actual work of setting boot parameters.
// Runtime-aware version.
func runCoreBootParamsSetWithRuntime(cmd *cobra.Command, opts *bootParamsSetOptions, bssClient *bss.BSSClient, rt *cli.Runtime) error {
	// Handle token for this command
	if err := rt.HandleToken(cmd); err != nil {
		return err
	}

	// The BSS BootParams struct we will send
	bp := opts.toBootParams()

	// Read payload from file first, allowing overwrites from flags
	if cmd.Flag("data").Changed {
		if err := rt.HandlePayload(cmd, &bp); err != nil {
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
	_, err := bssClient.PutBootParams(cmd.Context(), bp, rt.Token)
	if err != nil {
		return cli.ClassifyClientError(err, "BSS boot parameter request yielded unsuccessful HTTP response", "failed to set boot parameters in BSS")
	}

	return nil
}

// validateBootParamsSetFlags validates that the required flags are present.
func validateBootParamsSetFlags(cmd *cobra.Command, args []string) error {
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
		PreRunE: validateBootParamsSetFlags,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Create client to use for requests with runtime
			bssClient, err := bss_lib.GetClientWithRuntime(cmd, rt)
			if err != nil {
				return err
			}

			// Extract options from flags
			// Since flags are registered with the correct types on this command,
			// these Get* calls cannot fail, so we ignore errors with explicit comments
			opts := &bootParamsSetOptions{}
			if cmd.Flag("xname").Changed {
				opts.Xname, _ = cmd.Flags().GetStringSlice("xname") //nolint:errcheck // Flag registered with matching type, error impossible
			}
			if cmd.Flag("mac").Changed {
				opts.Mac, _ = cmd.Flags().GetStringSlice("mac") //nolint:errcheck // Flag registered with matching type, error impossible
			}
			if cmd.Flag("nid").Changed {
				opts.Nid, _ = cmd.Flags().GetInt32Slice("nid") //nolint:errcheck // Flag registered with matching type, error impossible
			}
			if cmd.Flag("kernel").Changed {
				opts.Kernel, _ = cmd.Flags().GetString("kernel") //nolint:errcheck // Flag registered with matching type, error impossible
			}
			if cmd.Flag("initrd").Changed {
				opts.Initrd, _ = cmd.Flags().GetString("initrd") //nolint:errcheck // Flag registered with matching type, error impossible
			}
			if cmd.Flag("params").Changed {
				opts.Params, _ = cmd.Flags().GetString("params") //nolint:errcheck // Flag registered with matching type, error impossible
			}

			return runCoreBootParamsSetWithRuntime(cmd, opts, bssClient, rt)
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
	bootParamsSetCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return bootParamsSetCmd
}
