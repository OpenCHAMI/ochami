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

// bssHostsGetCmd represents the bss-hosts-get command
var bssHostsGetCmd = &cobra.Command{
	Use:   "get",
	Args:  cobra.NoArgs,
	Short: "Get information on hosts known to BSS",
	Long: `Get information on hosts known to BSS.

See ochami-bss(1) for more details.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Without a base URI, we cannot do anything
		bssBaseURI, err := getBaseURIBSS(cmd)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for BSS")
			logHelpError(cmd)
			os.Exit(1)
		}

		// Create client to make request to BSS
		bssClient, err := bss.NewClient(bssBaseURI, insecure)
		if err != nil {
			log.Logger.Error().Err(err).Msg("error creating new BSS client")
			logHelpError(cmd)
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
				x, err := cmd.Flags().GetString("xname")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch xname")
					logHelpError(cmd)
					os.Exit(1)
				}
				values.Add("name", x)
			}
			if cmd.Flag("mac").Changed {
				m, err := cmd.Flags().GetString("mac")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch mac")
					logHelpError(cmd)
					os.Exit(1)
				}
				values.Add("mac", m)
			}
			if cmd.Flag("nid").Changed {
				n, err := cmd.Flags().GetInt32("nid")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch nid")
					logHelpError(cmd)
					os.Exit(1)
				}
				values.Add("nid", fmt.Sprintf("%d", n))
			}
			qstr = values.Encode()
		}
		httpEnv, err := bssClient.GetHosts(qstr)
		if err != nil {
			if errors.Is(err, client.UnsuccessfulHTTPError) {
				log.Logger.Error().Err(err).Msg("BSS hosts request yielded unsuccessful HTTP response")
			} else {
				log.Logger.Error().Err(err).Msg("failed to request hosts from BSS")
			}
			logHelpError(cmd)
			os.Exit(1)
		}

		// Print output
		outFmt, err := cmd.Flags().GetString("output-format")
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get value for --output-format")
			logHelpError(cmd)
			os.Exit(1)
		}
		if outBytes, err := client.FormatBody(httpEnv.Body, outFmt); err != nil {
			log.Logger.Error().Err(err).Msg("failed to format output")
			logHelpError(cmd)
			os.Exit(1)
		} else {
			fmt.Printf(string(outBytes))
		}
	},
}

func init() {
	bssHostsGetCmd.Flags().StringP("xname", "x", "", "xname whose host information to get")
	bssHostsGetCmd.Flags().StringP("mac", "m", "", "MAC address whose boot parameters to get")
	bssHostsGetCmd.Flags().Int32P("nid", "n", 0, "node ID whose host information to get")
	bssHostsGetCmd.Flags().StringP("output-format", "F", defaultOutputFormat, "format of output printed to standard output")
	bssHostsCmd.AddCommand(bssHostsGetCmd)
}
