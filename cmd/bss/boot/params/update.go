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
the data is read from standard input.

This command sends a PATCH to BSS. An access cli.Token is required.

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

			// Send 'em off
			_, err = bssClient.PatchBootParams(bp, cli.Token)
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
	bootParamsUpdateCmd.Flags().String("kernel", "", "URI of kernel")
	bootParamsUpdateCmd.Flags().String("initrd", "", "URI of initrd/initramfs")
	bootParamsUpdateCmd.Flags().String("params", "", "kernel parameters")
	bootParamsUpdateCmd.Flags().StringSliceP("xname", "x", []string{}, "one or more xnames whose boot parameters to update")
	bootParamsUpdateCmd.Flags().StringSliceP("mac", "m", []string{}, "one or more MAC addresses whose boot parameters to update")
	bootParamsUpdateCmd.Flags().Int32SliceP("nid", "n", []int32{}, "one or more node IDs whose boot parameters to update")
	bootParamsUpdateCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	bootParamsUpdateCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	bootParamsUpdateCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return bootParamsUpdateCmd
}
