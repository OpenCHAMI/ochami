// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/bss"
	"github.com/spf13/cobra"
)

// bootParamsGetCmd represents the bss-boot-params-get command
var bootParamsGetCmd = &cobra.Command{
	Use:   "get",
	Args:  cobra.NoArgs,
	Short: "Get boot parameters for one or all nodes",
	Long: `Get boot parameters for one or all nodes. If no options are passed, all boot
parameters are returned. Optionally, --mac, --xname, and/or --nid can be passed at least once
to get boot parameters for specific components.

This command sends a GET to BSS. An access token is required.`,
	Example: `  ochami bss boot params get
  ochami bss boot params get --mac 00:de:ad:be:ef:00
  ochami bss boot params get --mac 00:de:ad:be:ef:00,00:c0:ff:ee:00:00
  ochami bss boot params get --mac 00:de:ad:be:ef:00 --mac 00:c0:ff:ee:00:00`,
	Run: func(cmd *cobra.Command, args []string) {
		// Without a base URI, we cannot do anything
		bssBaseURI, err := getBaseURIBSS(cmd)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for BSS")
			os.Exit(1)
		}

		// This endpoint requires authentication, so a token is needed
		setTokenFromEnvVar(cmd)
		checkToken(cmd)

		// Create client to make request to BSS
		bssClient, err := bss.NewClient(bssBaseURI, insecure)
		if err != nil {
			log.Logger.Error().Err(err).Msg("error creating new BSS client")
			os.Exit(1)
		}

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(bssClient.OchamiClient)

		// If no ID flags are specified, get all boot parameters
		qstr := ""
		if cmd.Flag("xname").Changed ||
			cmd.Flag("mac").Changed ||
			cmd.Flag("nid").Changed {
			values := url.Values{}
			if cmd.Flag("xname").Changed {
				s, err := cmd.Flags().GetStringSlice("xname")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch xname list")
					os.Exit(1)
				}
				for _, x := range s {
					values.Add("name", x)
				}
			}
			if cmd.Flag("mac").Changed {
				s, err := cmd.Flags().GetStringSlice("mac")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch mac list")
					os.Exit(1)
				}
				for _, m := range s {
					values.Add("mac", m)
				}
			}
			if cmd.Flag("nid").Changed {
				s, err := cmd.Flags().GetInt32Slice("nid")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch nid list")
					os.Exit(1)
				}
				for _, n := range s {
					values.Add("nid", fmt.Sprintf("%d", n))
				}
			}
			qstr = values.Encode()
		}
		httpEnv, err := bssClient.GetBootParams(qstr, token)
		if err != nil {
			if errors.Is(err, client.UnsuccessfulHTTPError) {
				log.Logger.Error().Err(err).Msg("BSS boot parameter request yielded unsuccessful HTTP response")
			} else {
				log.Logger.Error().Err(err).Msg("failed to request boot parameters from BSS")
			}
			os.Exit(1)
		}

		outFmt, err := cmd.Flags().GetString("output-format")
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get value for --output-format")
			os.Exit(1)
		}
		if outBytes, err := client.FormatBody(httpEnv.Body, outFmt); err != nil {
			log.Logger.Error().Err(err).Msg("failed to format output")
			os.Exit(1)
		} else {
			fmt.Printf(string(outBytes))
		}
	},
}

func init() {
	bootParamsGetCmd.Flags().StringSliceP("xname", "x", []string{}, "one or more xnames whose boot parameters to get")
	bootParamsGetCmd.Flags().StringSliceP("mac", "m", []string{}, "one or more MAC addresses whose boot parameters to get")
	bootParamsGetCmd.Flags().Int32SliceP("nid", "n", []int32{}, "one or more node IDs whose boot parameters to get")
	bootParamsGetCmd.Flags().StringP("output-format", "F", defaultOutputFormat, "format of output printed to standard output")
	bootParamsCmd.AddCommand(bootParamsGetCmd)
}
