// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"os"

	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/ci"
	"github.com/spf13/cobra"
)

// cloudInitNodeSetCmd represents the "cloud-init node set" command
var cloudInitNodeSetCmd = &cobra.Command{
	Use:   "set [-d (<data> | @<path>)] [-f <format>]",
	Args:  cobra.NoArgs,
	Short: "Set cloud-init meta-data for specific nodes",
	Long: `Set cloud-init meta-data for specific nodes. Data is read from
standard input. Alternatively, pass -d to pass raw payload data
or (if flag argument starts with @) a file containing the payload
data. -f can be specified to change the format of the input
payload data ('json' by default), but the rules above still apply
for the payload. If "-" is used as the input payload filename, the
data is read from standard input.

See ochami-cloud-init(1) for more details.`,
	Example: `  # Set cloud-init node meta-data using input payload data
  ochami cloud-init node set -d '[{
    "availability-zone": "string",
    "cloud-init-base-url": "string",
    "cloud-provider": "string",
    "cluster-name": "demo",
    "hostname": "string",
    "id": "x3000c1b1n1",
    "instance-id": "string",
    "instance-type": "string",
    "local-hostname": "compute-1",
    "public-keys": [
      "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMLtQNuzGcMDatF+YVMMkuxbX2c5v2OxWftBhEVfFb+U user1@demo-head",
      "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIB4vVRvkzmGE5PyWX2fuzJEgEfET4PRLHXCnD1uFZ8ZL user2@demo-head"
    ],
    "region": "string"
  }]'

  # Set cloud-init node meta-data using input payload file
  ochami cloud-init group set -d @payload.json
  ochami cloud-init group set -d @payload.yaml -f yaml

  # Set cloud-init node meta-data using data from standard input
  echo '<json_data>' | ochami cloud-init group set
  echo '<json_data>' | ochami cloud-init group set -d @-
  echo '<yaml_data>' | ochami cloud-init group set -f yaml
  echo '<yaml_data>' | ochami cloud-init group set -d @- -f yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		// Without a base URI, we cannot do anything
		cloudInitbaseURI, err := getBaseURICloudInit(cmd)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for cloud-init")
			logHelpError(cmd)
			os.Exit(1)
		}

		// This endpoint requires authentication, so a token is needed
		setTokenFromEnvVar(cmd)
		checkToken(cmd)

		// Create client to make request to cloud-init
		cloudInitClient, err := ci.NewClient(cloudInitbaseURI, insecure)
		if err != nil {
			log.Logger.Error().Err(err).Msg("error creating new cloud-init client")
			logHelpError(cmd)
			os.Exit(1)
		}

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(cloudInitClient.OchamiClient)

		// The instance information list we will send
		ciInstInfo := []cistore.OpenCHAMIInstanceInfo{}

		// Read payload from file or stdin.
		if cmd.Flag("data").Changed {
			handlePayload(cmd, &ciInstInfo)
		} else {
			handlePayloadStdin(cmd, &ciInstInfo)
		}

		// Send data
		_, errs, err := cloudInitClient.PutInstanceInfo(ciInstInfo, token)
		if err != nil {
			log.Logger.Error().Err(err).Msgf("failed to set instance info")
			logHelpError(cmd)
			os.Exit(1)
		}
		// Since the requests are done iteratively, we need to deal with
		// each error that might have occurred.
		var errorsOccurred = false
		for _, err := range errs {
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("cloud-init node instance info request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to set node instance info in cloud-init")
				}
				errorsOccurred = true
			}
		}
		if errorsOccurred {
			log.Logger.Warn().Msg("cloud-init node instance info setting completed with errors")
			logHelpError(cmd)
			os.Exit(1)
		}
	},
}

func init() {
	cloudInitNodeSetCmd.Flags().VarP(&formatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")
	cloudInitNodeSetCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")

	cloudInitNodeSetCmd.RegisterFlagCompletionFunc("format-input", completionFormatData)

	cloudInitNodeCmd.AddCommand(cloudInitNodeSetCmd)
}
