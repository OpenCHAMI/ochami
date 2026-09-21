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
the data is read from standard input.

This command sends a DELETE to BSS. An access token is required.

See ochami-bss(1) for more details.`,
		Example: `  # Delete boot parameters using CLI flags
  ochami bss boot params delete --kernel https://example.com/kernel
  ochami bss boot params delete --kernel https://example.com/kernel --initrd https://example.com/initrd

  # Delete boot parameters using input payload data
  ochami bss boot params delete -d '{"macs":["00:de:ad:be:ef:00"]}'
  ochami bss boot params delete -d '{"kernel":"https://example.com/kernel"}'

  # Delete boot parameters using input payload data
  ochami bss boot params delete -d @payload.json
  ochami bss boot params delete -d @payload.yaml -f yaml

  # Delete boot parameters using data from standard input
  echo '<json_data>' | ochami bss boot params delete -d @-
  echo '<yaml_data>' | ochami bss boot params delete -d @- -f yaml`,
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
			// The BSS BootParams struct we will send
			bp := bssTypes.BootParams{}

			// Read payload from file first, allowing overwrites from flags
			if cmd.Flag("data").Changed {
				if err := cli.HandlePayload(cmd, &bp); err != nil {
					return err
				}
			}

			// Set the hosts the boot parameters are for
			var err error
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

			// Ask before attempting deletion unless --no-confirm was passed
			noConfirm, _ := cmd.Flags().GetBool("no-confirm") //nolint:errcheck // flag is registered with the matching type on this command
			if !noConfirm {
				log.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
				respDelete, err := cli.Ios.LoopYesNo("Really delete?")
				if err != nil {
					return cli.Errorf(cli.CodeGeneric, "error fetching user input: %w", err)
				} else if !respDelete {
					log.Logger.Info().Msg("User aborted boot parameter deletion")
					return nil
				} else {
					log.Logger.Debug().Msg("User answered affirmatively to delete boot parameters")
				}
			}

			// Create client to use for requests
			bssClient, err := bss_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// Send 'em off
			_, err = bssClient.DeleteBootParams(cmd.Context(), bp, cli.Token)
			if err != nil {
				return cli.ClassifyClientError(err, "BSS boot parameter request yielded unsuccessful HTTP response", "failed to set boot parameters in BSS")

			}

			return nil
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
	bootParamsDelete.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")
	bootParamsDelete.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return bootParamsDelete
}
