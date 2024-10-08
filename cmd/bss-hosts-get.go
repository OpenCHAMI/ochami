// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/cobra"
	"github.com/synackd/ochami/internal/client"
	"github.com/synackd/ochami/internal/log"
)

// bssHostsGetCmd represents the bss-hosts-get command
var bssHostsGetCmd = &cobra.Command{
	Use:   "get",
	Args:  cobra.NoArgs,
	Short: "Get information on hosts known to SMD",
	Run: func(cmd *cobra.Command, args []string) {
		// Without a base URI, we cannot do anything
		bssBaseURI, err := getBaseURI(cmd)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for BSS")
			os.Exit(1)
		}

		// Create client to make request to BSS
		bssClient, err := client.NewBSSClient(bssBaseURI, insecure)
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
				x, err := cmd.Flags().GetString("xname")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch xname")
					os.Exit(1)
				}
				values.Add("name", x)
			}
			if cmd.Flag("mac").Changed {
				m, err := cmd.Flags().GetString("mac")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch mac")
					os.Exit(1)
				}
				values.Add("mac", m)
			}
			if cmd.Flag("nid").Changed {
				n, err := cmd.Flags().GetInt32("nid")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch nid")
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
			os.Exit(1)
		}
		fmt.Println(string(httpEnv.Body))
	},
}

func init() {
	bssHostsGetCmd.Flags().StringP("xname", "x", "", "xname whose host information to get")
	bssHostsGetCmd.Flags().StringP("mac", "m", "", "MAC address whose boot parameters to get")
	bssHostsGetCmd.Flags().Int32P("nid", "n", 0, "node ID whose host information to get")
	bssHostsCmd.AddCommand(bssHostsGetCmd)
}
