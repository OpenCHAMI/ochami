// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client/ci"
	"github.com/spf13/cobra"
)

// cloudInitDefaultsSetCmd represents the cloud-init-defaults-set command
var cloudInitDefaultsSetCmd = &cobra.Command{
	Use:   "set [-d (<data> | @<path>)] [-f <format>]",
	Args:  cobra.NoArgs,
	Short: "Set default meta-data for cluster in cloud-init",
	Long: `Set default meta-data for cluster in cloud-init. Pass -d to pass raw payload
data or (if flag argument starts with @) a file containing the payload
data. -f can be specified to change the format of the input payload
data ('json' by default), but the rules above still apply for the
payload. If "-" is used as the input payload filename, the data is read
from standard input. If -d is not passed, the data is read from stdin.

See ochami-cloud-init(1) for more details.`,
	Example: `  # Set cloud-init defaults using input payload data
  ochami cloud-init defaults set -d '{
    "availability-zone": "string",
    "base-url": "http://demo.openchami.cluster:8081/cloud-init",
    "boot-subnet": "string",
    "cloud_provider": "string",
    "cluster-name": "demo",
    "nid-length": 3,
    "public-keys": [
      "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMLtQNuzGcMDatF+YVMMkuxbX2c5v2OxWftBhEVfFb+U user1@demo-head",
      "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIB4vVRvkzmGE5PyWX2fuzJEgEfET4PRLHXCnD1uFZ8ZL user2@demo-head"
    ],
    "region": "string",
    "short-name": "nid",
    "wg-subnet": "string"
  }'

  # Set cloud-init defaults using input payload file
  ochami cloud-init defaults set -d @payload.json
  ochami cloud-init defaults set -d @payload.yaml -f yaml

  # Set cloud-init defaults using data from standard input
  echo '<json_data>' | ochami cloud-init defaults set
  echo '<json_data>' | ochami cloud-init defaults set -d @-
  echo '<yaml_data>' | ochami cloud-init defaults set -f yaml
  echo '<yaml_data>' | ochami cloud-init defaults set -d @- -f yaml`,
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

		// The ClusterDefaults data we will send
		ciDflts := cistore.ClusterDefaults{}

		// Read payload from file or stdin.
		if cmd.Flag("data").Changed {
			handlePayload(cmd, &ciDflts)
		} else {
			handlePayloadStdin(cmd, &ciDflts)
		}

		// Send data
		_, err = cloudInitClient.PostDefaults(ciDflts, token)
		if err != nil {
			log.Logger.Error().Err(err).Msgf("failed to set defaults")
			logHelpError(cmd)
			os.Exit(1)
		}
	},
}

func init() {
	cloudInitDefaultsSetCmd.Flags().VarP(&formatInput, "format-input", "f", "format of input payload data (json,yaml)")
	cloudInitDefaultsSetCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")

	cloudInitDefaultsSetCmd.RegisterFlagCompletionFunc("format-input", completionFormatData)

	cloudInitDefaultsCmd.AddCommand(cloudInitDefaultsSetCmd)
}
