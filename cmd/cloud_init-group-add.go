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

// cloudInitGroupAddCmd represents the "cloud-init group add" command
var cloudInitGroupAddCmd = &cobra.Command{
	Use:   "add [-d (<data> | @<path>)] [-f <format>]",
	Args:  cobra.NoArgs,
	Short: "Add one or more new groups to cloud-init",
	Long: `Add one or more new groups to cloud-init. Data is read from
standard input. Alternatively, pass -d to pass raw payload data
or (if flag argument starts with @) a file containing the payload
data. -f can be specified to change the format of the input
payload data ('json' by default), but the rules above still apply
for the payload. If "-" is used as the input payload filename, the
data is read from standard input.

See ochami-cloud-init(1) for more details.`,
	Example: `  # Add cloud-init groups using input payload data
  ochami cloud-init group add -d '[{
    "description": "The compute group",
    "file": {
      "content": "IyMgdGVtcGxhdGU6IGppbmphCiNjbG91ZC1jb25maWcKbWVyZ2VfaG93OgotIG5hbWU6IGxpc3QKICBzZXR0aW5nczogW2FwcGVuZF0KLSBuYW1lOiBkaWN0CiAgc2V0dGluZ3M6IFtub19yZXBsYWNlLCByZWN1cnNlX2xpc3RdCnVzZXJzOgogIC0gbmFtZTogcm9vdAogICAgc3NoX2F1dGhvcml6ZWRfa2V5czoge3sgZHMubWV0YV9kYXRhLmluc3RhbmNlX2RhdGEudjEucHVibGljX2tleXMgfX0KZGlzYWJsZV9yb290OiBmYWxzZQo=",
      "encoding": "base64",
      "filename": "string"
    },
    "meta-data": {
      "foo": "bar"
    },
    "name": "compute"
  }]'

  # Add cloud-init groups using input payload file
  ochami cloud-init group add -d @payload.json
  ochami cloud-init group add -d @payload.yaml -f yaml

  # Add cloud-init groups using data from standard input
  echo '<json_data>' | ochami cloud-init group add
  echo '<json_data>' | ochami cloud-init group add -d @-
  echo '<yaml_data>' | ochami cloud-init group add -f yaml
  echo '<yaml_data>' | ochami cloud-init group add -d @- -f yaml`,
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

		// The group data we will send
		ciGroups := []cistore.GroupData{}

		// Read payload from file or stdin.
		if cmd.Flag("data").Changed {
			handlePayload(cmd, &ciGroups)
		} else {
			handlePayloadStdin(cmd, &ciGroups)
		}

		// Send data
		_, errs, err := cloudInitClient.PostGroups(ciGroups, token)
		if err != nil {
			log.Logger.Error().Err(err).Msgf("failed to add groups")
			logHelpError(cmd)
			os.Exit(1)
		}
		// Since the requests are done iteratively, we need to deal with
		// each error that might have occurred.
		var errorsOccurred = false
		for _, err := range errs {
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("cloud-init group request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to add new group to cloud-init")
				}
				errorsOccurred = true
			}
		}
		if errorsOccurred {
			log.Logger.Warn().Msg("cloud-init group addition completed with errors")
			logHelpError(cmd)
			os.Exit(1)
		}
	},
}

func init() {
	cloudInitGroupAddCmd.Flags().VarP(&formatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")
	cloudInitGroupAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")

	cloudInitGroupAddCmd.RegisterFlagCompletionFunc("format-input", completionFormatData)

	cloudInitGroupCmd.AddCommand(cloudInitGroupAddCmd)
}
