// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"os"

	"github.com/OpenCHAMI/bss/pkg/bssTypes"
	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
)

// bssBootParamsAddCmd represents the "bss boot params add" command
var bssBootParamsAddCmd = &cobra.Command{
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
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// cmd.LocalFlags().NFlag() doesn't seem to work, so we check every flag
		if len(args) == 0 &&
			!cmd.Flag("xname").Changed && !cmd.Flag("nid").Changed && !cmd.Flag("mac").Changed &&
			!cmd.Flag("kernel").Changed && !cmd.Flag("initrd").Changed && !cmd.Flag("data").Changed {
			printUsageHandleError(cmd)
			os.Exit(0)
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Create client to use for requests
		bssClient := bssGetClient(cmd, true)

		// The BSS BootParams struct we will send
		bp := bssTypes.BootParams{}

		// Read payload from file first, allowing overwrites from flags
		handlePayload(cmd, &bp)

		// Set the hosts the boot parameters are for
		var err error
		if cmd.Flag("xname").Changed {
			bp.Hosts, err = cmd.Flags().GetStringSlice("xname")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to fetch xname list")
				logHelpError(cmd)
				os.Exit(1)
			}
		}
		if cmd.Flag("mac").Changed {
			bp.Macs, err = cmd.Flags().GetStringSlice("mac")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to fetch mac list")
				logHelpError(cmd)
				os.Exit(1)
			}
			if err = bp.CheckMacs(); err != nil {
				log.Logger.Error().Err(err).Msg("invalid mac(s)")
				logHelpError(cmd)
				os.Exit(1)
			}
		}
		if cmd.Flag("nid").Changed {
			bp.Nids, err = cmd.Flags().GetInt32Slice("nid")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to fetch nid list")
				logHelpError(cmd)
				os.Exit(1)
			}
		}

		// Set the boot parameters
		if cmd.Flag("kernel").Changed {
			bp.Kernel, err = cmd.Flags().GetString("kernel")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to fetch kernel uri")
				logHelpError(cmd)
				os.Exit(1)
			}
		}
		if cmd.Flag("initrd").Changed {
			bp.Initrd, err = cmd.Flags().GetString("initrd")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to fetch initrd uri")
				logHelpError(cmd)
				os.Exit(1)
			}
		}
		if cmd.Flag("params").Changed {
			bp.Params, err = cmd.Flags().GetString("params")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to fetch params")
				logHelpError(cmd)
				os.Exit(1)
			}
		}

		// Send 'em off
		_, err = bssClient.PostBootParams(bp, token)
		if err != nil {
			if errors.Is(err, client.UnsuccessfulHTTPError) {
				log.Logger.Error().Err(err).Msg("BSS boot parameter request yielded unsuccessful HTTP response")
			} else {
				log.Logger.Error().Err(err).Msg("failed to add boot parameters to BSS")
			}
			logHelpError(cmd)
			os.Exit(1)
		}
	},
}

func init() {
	bssBootParamsAddCmd.Flags().String("kernel", "", "URI of kernel")
	bssBootParamsAddCmd.Flags().String("initrd", "", "URI of initrd/initramfs")
	bssBootParamsAddCmd.Flags().String("params", "", "kernel parameters")
	bssBootParamsAddCmd.Flags().StringSliceP("xname", "x", []string{}, "one or more xnames whose boot parameters to add")
	bssBootParamsAddCmd.Flags().StringSliceP("mac", "m", []string{}, "one or more MAC addresses whose boot parameters to add")
	bssBootParamsAddCmd.Flags().Int32SliceP("nid", "n", []int32{}, "one or more node IDs whose boot parameters to add")
	bssBootParamsAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	bssBootParamsAddCmd.Flags().VarP(&formatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	bssBootParamsAddCmd.RegisterFlagCompletionFunc("format-input", completionFormatData)
	bssBootParamsAddCmd.MarkFlagsOneRequired("xname", "mac", "nid", "data")
	bssBootParamsAddCmd.MarkFlagsOneRequired("kernel", "initrd", "params", "data")

	bssBootParamsCmd.AddCommand(bssBootParamsAddCmd)
}
