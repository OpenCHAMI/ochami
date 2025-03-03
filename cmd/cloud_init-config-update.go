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

// cloudInitConfigUpdateCmd represents the cloud-init-config-update command
var cloudInitConfigUpdateCmd = &cobra.Command{
	Use:   "update (-f <payload_file> | -d <payload_data>)",
	Args:  cobra.NoArgs,
	Short: "Update cloud-init config for one or more ids, overwriting any previous",
	Long: `Update cloud-init config for one or more ids. Either a payload file
containing the data or the JSON data itself must be passed. Data
is represented by a JSON array of cloud-init configs, even if only
one is being passed. An alternative to using -d would be to use -f
and passing -, which will cause ochami to read the data from
standard input.

This command sends a PUT to cloud-init.`,
	Example: `  ochami cloud-init config update -d \
    '[ \
       { \
         "name": "compute", \
         "cloud-init": { \
           "userdata": { \
             "write_files": [ \
               { \
                 "content": "new hello world content",
                 "path": "/etc/hello"
               } \
             ] \
           } \
         } \
       } \
     ]'
  ochami cloud-init config update -f payload.json
  ochami cloud-init config update -f payload.yaml --payload-format yaml
  echo '<json_data>' | ochami cloud-init config update -f -
  echo '<yaml_data>' | ochami cloud-init config update -f - --payload-format yaml`,
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
			_, errs, err = cloudInitClient.PutConfigsSecure(ciData, token)
		} else {
			_, errs, err = cloudInitClient.PutConfigs(ciData, token)
		}
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to set cloud-init configs")
			os.Exit(1)
		}
		// Since cloudInitClient.Put* functions do the setting iteratively, we need to deal with
		// each error that might have occurred.
		var errorsOccurred = false
		for _, e := range errs {
			if e != nil {
				if errors.Is(e, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(e).Msg("cloud-init config request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(e).Msg("failed to set config(s) in cloud-init")
				}
				errorsOccurred = true
			}
		}
		// Warn the user if any errors occurred during editing iterations
		if errorsOccurred {
			log.Logger.Warn().Msgf("cloud-init config setting completed with errors")
			os.Exit(1)
		}
	},
}

func init() {
	cloudInitConfigUpdateCmd.Flags().StringP("data", "d", "", "raw JSON data to use as payload")
	cloudInitConfigUpdateCmd.Flags().StringP("payload", "f", "", "file containing the request payload; JSON format unless --payload-format specified")
	cloudInitConfigUpdateCmd.Flags().StringP("payload-format", "F", defaultPayloadFormat, "format of payload file (yaml,json) passed with --payload")

	cloudInitConfigUpdateCmd.MarkFlagsMutuallyExclusive("data", "payload")
	cloudInitConfigUpdateCmd.MarkFlagsMutuallyExclusive("data", "payload-format")
	cloudInitConfigUpdateCmd.MarkFlagsOneRequired("data", "payload")

	cloudInitConfigCmd.AddCommand(cloudInitConfigUpdateCmd)
}
