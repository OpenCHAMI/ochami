// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/OpenCHAMI/cloud-init/pkg/citypes"
	"github.com/OpenCHAMI/ochami/internal/config"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/ci"
	"github.com/spf13/cobra"
)

// cloudInitConfigAddCmd represents the cloud-init-config-add command
var cloudInitConfigAddCmd = &cobra.Command{
	Use:   "add (-f <payload_file> | -d <json_data>)",
	Args:  cobra.NoArgs,
	Short: "Add one or more new cloud-init configs",
	Long: `Add one or more new cloud-init configs. Either a payload file
containing the data or the JSON data itself must be passed.
Data is represented by a JSON array of cloud-init configs,
even if only one is being passed. An alternative to using
-d would be to use -f and passing -, which will cause ochami
to read the data from standard input.

This command sends a POST to cloud-init.`,
	Example: `  ochami cloud-init config add -d \
    '[ \
       { \
         "name": "compute", \
         "cloud-init": { \
           "userdata": { \
             "write_files": [ \
               { \
                 "content": "hello world",
                 "path": "/etc/hello"
               } \
             ] \
           }, \
           "metadata": { \
             "instance-id": "compute"
           } \
         } \
       } \
     ]'
  ochami cloud-init config add -f payload.json
  ochami cloud-init config add -f payload.yaml --payload-format yaml
  echo '<json_data>' | ochami cloud-init config add -f -
  echo '<yaml_data>' | ochami cloud-init config add -f - --payload-format yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		// Without a base URI, we cannot do anything
		cloudInitBaseURI, err := getBaseURI(cmd, config.ServiceCloudInit)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for cloud-init")
			os.Exit(1)
		}

		// This endpoint requires authentication, so a token is needed
		setTokenFromEnvVar(cmd)
		checkToken(cmd)

		// Create client to make request to cloud-init
		cloudInitClient, err := ci.NewClient(cloudInitBaseURI, insecure)
		if err != nil {
			log.Logger.Error().Err(err).Msg("error creating new cloud-init client")
			os.Exit(1)
		}

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(cloudInitClient.OchamiClient)

		var ciData []citypes.CI
		if cmd.Flag("payload").Changed {
			// Use payload file if passed
			handlePayload(cmd, &ciData)
		} else if cmd.Flag("data").Changed {
			// ...otherwise try to read raw JSON from CLI
			rawJSON, err := cmd.Flags().GetString("data")
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to fetch json data")
				os.Exit(1)
			}
			err = json.Unmarshal([]byte(rawJSON), &ciData)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to marshal json data")
				os.Exit(1)
			}
		}

		// Send off request
		var errs []error
		if cloudInitCmd.Flag("secure").Changed {
			_, errs, err = cloudInitClient.PostConfigsSecure(ciData, token)
		} else {
			_, errs, err = cloudInitClient.PostConfigs(ciData, token)
		}
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to add cloud-init configs")
			os.Exit(1)
		}
		// Since cloudInitClient.Post* functions do the addition iteratively, we need to deal with
		// each error that might have occurred.
		var errorsOccurred = false
		for _, e := range errs {
			if e != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(e).Msg("cloud-init config request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(e).Msg("failed to add config(s) to cloud-init")
				}
				errorsOccurred = true
			}
		}
		// Warn the user if any errors occurred during addition iterations
		if errorsOccurred {
			log.Logger.Warn().Msg("cloud-init config addition completed with errors")
			os.Exit(1)
		}
	},
}

func init() {
	cloudInitConfigAddCmd.Flags().StringP("data", "d", "", "raw JSON data to use as payload")
	cloudInitConfigAddCmd.Flags().StringP("payload", "f", "", "file containing the request payload; JSON format unless --payload-format specified")
	cloudInitConfigAddCmd.Flags().StringP("payload-format", "F", defaultPayloadFormat, "format of payload file (yaml,json) passed with --payload")

	cloudInitConfigAddCmd.MarkFlagsMutuallyExclusive("data", "payload")
	cloudInitConfigAddCmd.MarkFlagsMutuallyExclusive("data", "payload-format")
	cloudInitConfigAddCmd.MarkFlagsOneRequired("data", "payload")

	cloudInitConfigCmd.AddCommand(cloudInitConfigAddCmd)
}
