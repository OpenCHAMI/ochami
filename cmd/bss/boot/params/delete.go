// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package params

import (
	"errors"
	"fmt"
	"os"

	"github.com/OpenCHAMI/bss/pkg/bssTypes"
	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"

	bss_lib "github.com/OpenCHAMI/ochami/internal/cli/bss"
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

This command sends a DELETE to BSS. An access cli.Token is required.

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
					return fmt.Errorf("expected -d or one of --xname, --nid, or --mac")
				} else if !anyChanged("kernel", "initrd", "params") {
					return fmt.Errorf("specifying any of --xname, --nid, or --mac also requires specifying at least one of --kernel, --initrd, or --params")
				}
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			bssClient := bss_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// The BSS BootParams struct we will send
			bp := bssTypes.BootParams{}

			// Read payload from file first, allowing overwrites from flags
			cli.HandlePayload(cmd, &bp)

			// Set the hosts the boot parameters are for
			var err error
			if cmd.Flag("xname").Changed {
				bp.Hosts, err = cmd.Flags().GetStringSlice("xname")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch xname list")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}
			}
			if cmd.Flag("mac").Changed {
				bp.Macs, err = cmd.Flags().GetStringSlice("mac")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch mac list")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}
				if err = bp.CheckMacs(); err != nil {
					log.Logger.Error().Err(err).Msg("invalid mac(s)")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}
			}
			if cmd.Flag("nid").Changed {
				bp.Nids, err = cmd.Flags().GetInt32Slice("nid")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch nid list")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}
			}

			// Set the boot parameters
			if cmd.Flag("kernel").Changed {
				bp.Kernel, err = cmd.Flags().GetString("kernel")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch kernel uri")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}
			}
			if cmd.Flag("initrd").Changed {
				bp.Initrd, err = cmd.Flags().GetString("initrd")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch initrd uri")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}
			}
			if cmd.Flag("params").Changed {
				bp.Params, err = cmd.Flags().GetString("params")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch params")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}
			}

			// Ask before attempting deletion unless --no-confirm was passed
			if !cmd.Flag("no-confirm").Changed {
				log.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
				respDelete, err := cli.Ios.LoopYesNo("Really delete?")
				if err != nil {
					log.Logger.Error().Err(err).Msg("Error fetching user input")
					os.Exit(1)
				} else if !respDelete {
					log.Logger.Info().Msg("User aborted boot parameter deletion")
					os.Exit(0)
				} else {
					log.Logger.Debug().Msg("User answered affirmatively to delete boot parameters")
				}
			}

			// Send 'em off
			_, err = bssClient.DeleteBootParams(bp, cli.Token)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("BSS boot parameter request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to set boot parameters in BSS")
				}
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
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
