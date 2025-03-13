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

// bootScriptGetCmd represents the bss-boot-script-get command
var bootScriptGetCmd = &cobra.Command{
	Use:   "get",
	Args:  cobra.NoArgs,
	Short: "Get iPXE boot script for a component",
	Long: `Get iPXE boot script for a component. Specifying one of --mac, --xname,
or --nid is required to specify which component to fetch the boot script for.

This command sends a GET to BSS. An access token is not required.

See ochami-bss(1) for more details.`,
	Example: `  ochami boot script get --mac 00:c0:ff:ee:00:00`,
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

		// Structure representing the boot script query string
		values := url.Values{}

		// At least one of these required
		if cmd.Flag("xname").Changed {
			s, err := cmd.Flags().GetStringSlice("xname")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to fetch xname list")
				logHelpError(cmd)
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
				logHelpError(cmd)
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
				logHelpError(cmd)
				os.Exit(1)
			}
			for _, n := range s {
				values.Add("nid", fmt.Sprintf("%d", n))
			}
		}

		// These are optional
		if cmd.Flag("retry").Changed {
			s, err := cmd.Flags().GetInt("retry")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to fetch number of retries")
				logHelpError(cmd)
				os.Exit(1)
			}
			values.Add("retry", fmt.Sprintf("%d", s))
		}
		if cmd.Flag("arch").Changed {
			s, err := cmd.Flags().GetString("arch")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to fetch arch")
				logHelpError(cmd)
				os.Exit(1)
			}
			values.Add("arch", s)
		}
		if cmd.Flag("timestamp").Changed {
			s, err := cmd.Flags().GetInt("timestamp")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to fetch timestamp")
				logHelpError(cmd)
				os.Exit(1)
			}
			values.Add("timestamp", fmt.Sprintf("%d", s))
		}
		qstr := values.Encode()

		httpEnv, err := bssClient.GetBootScript(qstr)
		if err != nil {
			if errors.Is(err, client.UnsuccessfulHTTPError) {
				log.Logger.Error().Err(err).Msg("BSS boot script request yielded unsuccessful HTTP response")
			} else {
				log.Logger.Error().Err(err).Msg("failed to request boot script from BSS")
			}
			logHelpError(cmd)
			os.Exit(1)
		}
		fmt.Println(string(httpEnv.Body))
	},
}

func init() {
	bootScriptGetCmd.Flags().StringSliceP("xname", "x", []string{}, "one or more xnames whose boot script to get")
	bootScriptGetCmd.Flags().StringSliceP("mac", "m", []string{}, "one or more MAC addresses whose boot script to get")
	bootScriptGetCmd.Flags().Int32SliceP("nid", "n", []int32{}, "one or more node IDs whose boot script to get")
	bootScriptGetCmd.Flags().Int("retry", 0, "number of times to retry fetching boot script on failed boot")
	bootScriptGetCmd.Flags().String("arch", "", "architecture value from iPXE variable ${buildarch}")
	bootScriptGetCmd.Flags().Int("timestamp", 0, "timestamp in seconds since Unix epoch for when SMD state needs to be updated by")

	bootScriptGetCmd.MarkFlagsOneRequired("xname", "mac", "nid")

	bootScriptCmd.AddCommand(bootScriptGetCmd)
}
