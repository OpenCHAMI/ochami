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

// cloudInitConfigGetCmd represents the cloud-init-config-get command
var cloudInitConfigGetCmd = &cobra.Command{
	Use:   "get [id]",
	Args:  cobra.MaximumNArgs(1),
	Short: "Get cloud-init configs, all or for an identifier",
	Example: `ochami cloud-init config get
  ochami cloud-init config get compute
  ochami cloud-init config get --secure compute`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// First and foremost, make sure config is loaded and logging
		// works.
		initConfigAndLogging(cmd, true)

		return nil
	},
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

		// Make requests
		var httpEnv client.HTTPEnvelope
		var id string
		if len(args) > 0 {
			id = args[0]
		}
		if cloudInitCmd.Flag("secure").Changed {
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

		// Format output
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
	cloudInitConfigGetCmd.Flags().StringP("output-format", "F", defaultOutputFormat, "format of output printed to standard output")
	cloudInitConfigCmd.AddCommand(cloudInitConfigGetCmd)
}
