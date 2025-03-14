// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/OpenCHAMI/ochami/internal/config"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/ci"
	"github.com/spf13/cobra"
)

// cloudInitDataGetCmd represents the cloud-init-data-get command
var cloudInitDataGetCmd = &cobra.Command{
	Use:   "get [--user | --meta | --vendor] <id>...",
	Args:  cobra.MinimumNArgs(1),
	Short: "Get cloud-init data for an identifier",
	Long: `Get cloud-init data for an identifier. By default, user-data is
retrieved. This also occurs if --user is passed. --meta or
--vendor can also be specified to fetch cloud-init meta-data
or vendor-data, respectively.`,
	Example: `  ochami cloud-init data get compute
  ochami cloud-init data get --user compute
  ochami cloud-init data get --meta compute
  ochami cloud-init data get --vendor compute`,
	Run: func(cmd *cobra.Command, args []string) {
		// Without a base URI, we cannot do anything
		cloudInitbaseURI, err := getBaseURI(cmd, config.ServiceCloudInit)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for cloud-init")
			os.Exit(1)
		}

		// Create client to make request to cloud-init
		cloudInitClient, err := ci.NewClient(cloudInitbaseURI, insecure)
		if err != nil {
			log.Logger.Error().Err(err).Msg("error creating new cloud-init client")
			os.Exit(1)
		}

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(cloudInitClient.OchamiClient)

		var (
			henvs  []client.HTTPEnvelope
			errs   []error
			ciType ci.CIDataType
		)

		if cmd.Flag("meta").Changed {
			ciType = ci.CloudInitMetaData
		} else if cmd.Flag("vendor").Changed {
			ciType = ci.CloudInitVendorData
		} else {
			ciType = ci.CloudInitUserData
		}

		if cloudInitCmd.Flag("secure").Changed {
			// This endpoint requires authentication, so a token is needed
			setTokenFromEnvVar(cmd)
			checkToken(cmd)

			henvs, errs, err = cloudInitClient.GetCloudInitDataSecure(ciType, args, token)
		} else {
			henvs, errs, err = cloudInitClient.GetCloudInitData(ciType, args)
		}

		if err != nil {
			log.Logger.Error().Err(err).Msgf("failed to get %s from cloud-init", ciType)
			os.Exit(1)
		}
		// Since the cloud-init data get functions do the deletion
		// iteratively, we need to deal with each error that might have
		// occurred.
		var errorsOccurred = false
		for _, e := range errs {
			if err != nil {
				if errors.Is(e, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(e).Msgf("cloud-init %s get yielded unsuccessful HTTP response", ciType)
				} else if e != nil {
					log.Logger.Error().Err(e).Msgf("failed to get %s", ciType)
				}
				errorsOccurred = true
			}
		}
		// Warn the user if any errors occurred during deletion iterations
		if errorsOccurred {
			log.Logger.Warn().Msgf("cloud-init %s get completed with errors", ciType)
			os.Exit(1)
		}

		// Print output
		for hidx, henv := range henvs {
			if hidx >= len(args) {
				log.Logger.Warn().Msgf("unknown cloud-init %s data found", ciType)
			} else {
				log.Logger.Info().Msgf("printing cloud-init %s for %s", ciType, args[hidx])
			}
			fmt.Printf(string(henv.Body))
		}
	},
}

func init() {
	cloudInitDataGetCmd.Flags().Bool("user", false, "fetch user-data")
	cloudInitDataGetCmd.Flags().Bool("meta", false, "fetch meta-data")
	cloudInitDataGetCmd.Flags().Bool("vendor", false, "fetch vendor-data")

	cloudInitDataGetCmd.MarkFlagsMutuallyExclusive("user", "meta", "vendor")

	cloudInitDataCmd.AddCommand(cloudInitDataGetCmd)
}
