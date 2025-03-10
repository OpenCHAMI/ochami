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
	Use:   "add (-d (<payload_data> | @<payload_file>))",
	Args:  cobra.NoArgs,
	Short: "Add one or more new cloud-init configs",
	Long: `Add one or more new cloud-init configs. Data is
represented by a JSON array of cloud-init configs,
even if only one is being passed.

This command sends a POST to cloud-init.

See ochami-cloud-init(1) for more details.`,
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
  ochami cloud-init config add -d @payload.json
  ochami cloud-init config add -d @payload.yaml -f yaml
  echo '<json_data>' | ochami cloud-init config add -d @-
  echo '<yaml_data>' | ochami cloud-init config add -d @- -f yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		// Without a base URI, we cannot do anything
		cloudInitBaseURI, err := getBaseURI(cmd, config.ServiceCloudInit)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for cloud-init")
			logHelpError(cmd)
			os.Exit(1)
		}

		// This endpoint requires authentication, so a token is needed
		setTokenFromEnvVar(cmd)
		checkToken(cmd)

		// Create client to make request to cloud-init
		cloudInitClient, err := ci.NewClient(cloudInitBaseURI, insecure)
		if err != nil {
			log.Logger.Error().Err(err).Msg("error creating new cloud-init client")
			logHelpError(cmd)
			os.Exit(1)
		}

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(cloudInitClient.OchamiClient)

		var ciData []citypes.CI
		if cmd.Flag("data").Changed {
			// Use payload file if passed
			handlePayload(cmd, &ciData)
		} else if cmd.Flag("data").Changed {
			// ...otherwise try to read raw JSON from CLI
			rawJSON, err := cmd.Flags().GetString("data")
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to fetch json data")
				logHelpError(cmd)
				os.Exit(1)
			}
			err = json.Unmarshal([]byte(rawJSON), &ciData)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to marshal json data")
				logHelpError(cmd)
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
			logHelpError(cmd)
			os.Exit(1)
		}
		// Since cloudInitClient.Post* functions do the addition iteratively, we need to deal with
		// each error that might have occurred.
		var errorsOccurred = false
		for _, e := range errs {
			if e != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(e).Msg("cloud-init config request yielded unsuccessful HTTP response")
					logHelpError(cmd)
				} else {
					log.Logger.Error().Err(e).Msg("failed to add config(s) to cloud-init")
					logHelpError(cmd)
				}
				errorsOccurred = true
			}
		}
		// Warn the user if any errors occurred during addition iterations
		if errorsOccurred {
			log.Logger.Warn().Msg("cloud-init config addition completed with errors")
			logHelpError(cmd)
			os.Exit(1)
		}
	},
}

func init() {
	cloudInitConfigAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	cloudInitConfigAddCmd.Flags().StringP("format-input", "f", defaultInputFormat, "format of input payload data (json,yaml)")

	cloudInitConfigAddCmd.MarkFlagsOneRequired("data")

	cloudInitConfigCmd.AddCommand(cloudInitConfigAddCmd)
}
