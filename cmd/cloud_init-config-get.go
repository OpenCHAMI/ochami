// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/OpenCHAMI/ochami/internal/client"
	"github.com/OpenCHAMI/ochami/internal/log"
)

// cloudInitConfigGetCmd represents the cloud-init-config-get command
var cloudInitConfigGetCmd = &cobra.Command{
	Use:   "get [id]",
	Args:  cobra.MaximumNArgs(1),
	Short: "Get cloud-init configs, all or for an identifier",
	Example: `ochami cloud-init config get
  ochami cloud-init config get compute
  ochami cloud-init config get --secure compute`,
	Run: func(cmd *cobra.Command, args []string) {
		// Without a base URI, we cannot do anything
		cloudInitbaseURI, err := getBaseURI(cmd)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for cloud-init")
			os.Exit(1)
		}

		// Create client to make request to cloud-init
		cloudInitClient, err := client.NewCloudInitClient(cloudInitbaseURI, insecure)
		if err != nil {
			log.Logger.Error().Err(err).Msg("error creating new cloud-init client")
			os.Exit(1)
		}

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(cloudInitClient.OchamiClient)

		var httpEnv client.HTTPEnvelope
		var id string
		if len(args) > 0 {
			id = args[0]
		}
		if cmd.Flag("secure").Changed {
			// This endpoint requires authentication, so a token is needed
			setTokenFromEnvVar(cmd)
			checkToken(cmd)

			httpEnv, err = cloudInitClient.GetConfigsSecure(id, token)
		} else {
			httpEnv, err = cloudInitClient.GetConfigs(id)
		}
		if err != nil {
			if errors.Is(err, client.UnsuccessfulHTTPError) {
				log.Logger.Error().Err(err).Msg("cloud-init config request yielded unsuccessful HTTP response")
			} else {
				log.Logger.Error().Err(err).Msg("failed to request configs from cloud-init")
			}
			os.Exit(1)
		}
		fmt.Println(string(httpEnv.Body))
	},
}

func init() {
	cloudInitConfigGetCmd.Flags().BoolP("secure", "s", false, "use secure cloud-init endpoint (token required)")
	cloudInitConfigCmd.AddCommand(cloudInitConfigGetCmd)
}
