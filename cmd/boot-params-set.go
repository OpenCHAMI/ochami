// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"os"

	"github.com/OpenCHAMI/bss/pkg/bssTypes"
	"github.com/spf13/cobra"
	"github.com/OpenCHAMI/ochami/internal/client"
	"github.com/OpenCHAMI/ochami/internal/log"
)

// bootParamsSetCmd represents the boot-params-set command
var bootParamsSetCmd = &cobra.Command{
	Use:   "set",
	Args:  cobra.NoArgs,
	Short: "Set boot parameters for one or more components, overwriting any previous",
	Long: `Set boot parameters for one or mote components, overwriting any previously-set
parameters. At least one of --kernel, --initrd, or --params is required to
tell ochami which boot data to set. Also, at least one of --xname, --mac,
or --nid is required to tell ochami which components need modification. Alternatively, pass -f
to pass a file (optionally specifying --payload-format, JSON by default), but the rules above
still apply for the payload.

This command sends a PUT to BSS. An access token is required.`,
	Example: `  ochami boot params set --xname x1000c1s7b0 --kernel https://example.com/kernel
  ochami boot params set --xname x1000c1s7b0,x1000c1s7b1 --kernel https://example.com/kernel
  ochami boot params set --xname x1000c1s7b0 --xname x1000c1s7b1 --kernel https://example.com/kernel
  ochami boot params set --xname x1000c1s7b0 --nid 1 --mac 00:c0:ff:ee:00:00 --params 'quiet nosplash'
  ochami boot params set -f payload.json
  ochami boot params set -f payload.yaml --payload-format yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		// cmd.LocalFlags().NFlag() doesn't seem to work, so we check every flag
		if len(args) == 0 &&
			!cmd.Flag("xname").Changed && !cmd.Flag("nid").Changed && !cmd.Flag("mac").Changed &&
			!cmd.Flag("kernel").Changed && !cmd.Flag("initrd").Changed && !cmd.Flag("payload").Changed {
			err := cmd.Usage()
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to print usage")
				os.Exit(1)
			}
			os.Exit(0)
		}

		// Without a base URI, we cannot do anything
		bssBaseURI, err := getBaseURI(cmd)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for BSS")
			os.Exit(1)
		}

		// This endpoint requires authentication, so a token is needed
		setTokenFromEnvVar(cmd)
		checkToken(cmd)

		// Create client to make request to BSS
		bssClient, err := client.NewBSSClient(bssBaseURI, insecure)
		if err != nil {
			log.Logger.Error().Err(err).Msg("error creating new BSS client")
			os.Exit(1)
		}

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(bssClient.OchamiClient)

		// The BSS BootParams struct we will send
		bp := bssTypes.BootParams{}

		// Read payload from file first, allowing overwrites from flags
		if cmd.Flag("payload").Changed {
			dFile := cmd.Flag("payload").Value.String()
			dFormat := cmd.Flag("payload-format").Value.String()
			err := client.ReadPayload(dFile, dFormat, &bp)
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to read payload for request")
				os.Exit(1)
			}
		}

		// Set the hosts the boot parameters are for
		if cmd.Flag("xname").Changed {
			bp.Hosts, err = cmd.Flags().GetStringSlice("xname")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to fetch xname list")
				os.Exit(1)
			}
		}
		if cmd.Flag("mac").Changed {
			bp.Macs, err = cmd.Flags().GetStringSlice("mac")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to fetch mac list")
				os.Exit(1)
			}
			if err = bp.CheckMacs(); err != nil {
				log.Logger.Error().Err(err).Msg("invalid mac(s)")
				os.Exit(1)
			}
		}
		if cmd.Flag("nid").Changed {
			bp.Nids, err = cmd.Flags().GetInt32Slice("nid")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to fetch nid list")
				os.Exit(1)
			}
		}

		// Set the boot parameters
		if cmd.Flag("kernel").Changed {
			bp.Kernel, err = cmd.Flags().GetString("kernel")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to fetch kernel uri")
				os.Exit(1)
			}
		}
		if cmd.Flag("initrd").Changed {
			bp.Initrd, err = cmd.Flags().GetString("initrd")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to fetch initrd uri")
				os.Exit(1)
			}
		}
		if cmd.Flag("params").Changed {
			bp.Params, err = cmd.Flags().GetString("params")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to fetch params")
				os.Exit(1)
			}
		}

		// Send 'em off
		_, err = bssClient.PutBootParams(bp, token)
		if err != nil {
			if errors.Is(err, client.UnsuccessfulHTTPError) {
				log.Logger.Error().Err(err).Msg("BSS boot parameter request yielded unsuccessful HTTP response")
			} else {
				log.Logger.Error().Err(err).Msg("failed to set boot parameters in BSS")
			}
			os.Exit(1)
		}
	},
}

func init() {
	bootParamsSetCmd.Flags().String("kernel", "", "URI of kernel")
	bootParamsSetCmd.Flags().String("initrd", "", "URI of initrd/initramfs")
	bootParamsSetCmd.Flags().String("params", "", "kernel parameters")
	bootParamsSetCmd.Flags().StringSliceP("xname", "x", []string{}, "one or more xnames whose boot parameters to set")
	bootParamsSetCmd.Flags().StringSliceP("mac", "m", []string{}, "one or more MAC addresses whose boot parameters to set")
	bootParamsSetCmd.Flags().Int32SliceP("nid", "n", []int32{}, "one or more node IDs whose boot parameters to set")
	bootParamsSetCmd.Flags().StringP("payload", "f", "", "file containing the request payload; JSON format unless --payload-format specified")
	bootParamsSetCmd.Flags().String("payload-format", defaultPayloadFormat, "format of payload file (yaml,json) passed with --payload")

	bootParamsSetCmd.MarkFlagsOneRequired("xname", "mac", "nid", "payload")
	bootParamsSetCmd.MarkFlagsOneRequired("kernel", "initrd", "params", "payload")

	bootParamsCmd.AddCommand(bootParamsSetCmd)
}
