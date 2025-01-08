// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"os"

	"github.com/OpenCHAMI/ochami/internal/client"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/spf13/cobra"
)

// rfeDeleteCmd represents the smd-rfe-delete command
var rfeDeleteCmd = &cobra.Command{
	Use:   "delete -f <payload_file> | --all | <xname>...",
	Short: "Delete one or more redfish endpoints",
	Long: `Delete one or more redfish endpoints. These can be specified by one or more xnames.
Alternatively, use -f to read the payload data from a file (optionally
specifying --payload-format, JSON by default). If - is used as the
argument to -f, the data is read from standard input.

This command sends a DELETE to SMD. An access token is required.`,
	Example: `  ochami smd rfe delete x3000c1s7b56
  ochami smd rfe delete x3000c1s7b56 x3000c1s7b56
  ochami smd rfe delete --all
  ochami smd rfe delete -f payload.json
  ochami smd rfe delete -f payload.yaml --payload-format yaml
  echo '<json_data>' | ochami smd rfe delete -f -
  echo '<yaml_data>' | ochami smd rfe delete -f - --payload-format yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		// With options, only one of:
		// - A payload file with -f
		// - --all
		// - A set of one or more xnames
		// must be passed.
		if len(args) == 0 {
			if !cmd.Flag("all").Changed && !cmd.Flag("payload").Changed {
				err := cmd.Usage()
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to print usage")
					os.Exit(1)
				}
				os.Exit(0)
			}
		}

		// Without a base URI, we cannot do anything
		smdBaseURI, err := getBaseURI(cmd)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for SMD")
			os.Exit(1)
		}

		// This endpoint requires authentication, so a token is needed
		setTokenFromEnvVar(cmd)
		checkToken(cmd)

		// Create client to make request to SMD
		smdClient, err := client.NewSMDClient(smdBaseURI, insecure)
		if err != nil {
			log.Logger.Error().Err(err).Msg("error creating new SMD client")
			os.Exit(1)
		}

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(smdClient.OchamiClient)

		// Ask before attempting deletion unless --force was passed
		if !cmd.Flag("force").Changed {
			log.Logger.Debug().Msg("--force not passed, prompting user to confirm deletion")
			var respDelete bool
			if cmd.Flag("all").Changed {
				respDelete = loopYesNo("Really delete ALL REDFISH ENDPOINTS?")
			} else {
				respDelete = loopYesNo("Really delete?")
			}
			if !respDelete {
				log.Logger.Info().Msg("User aborted redfish endpoint deletion")
				os.Exit(0)
			} else {
				log.Logger.Debug().Msg("User answered affirmatively to delete redfish endpoints")
			}
		}

		// Create list of xnames to delete
		var rfeSlice client.RedfishEndpointSlice
		var xnameSlice []string
		if cmd.Flag("payload").Changed {
			// Use payload file if passed
			handlePayload(cmd, rfeSlice.RedfishEndpoints)
		} else {
			// ...otherwise, use passed CLI arguments
			xnameSlice = args
		}

		// Perform deletion
		if cmd.Flag("all").Changed {
			// If --all passed, we don't care about any passed arguments
			_, err := smdClient.DeleteRedfishEndpointsAll(token)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("SMD redfish endpoint deletion yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to delete redfish endpoints in SMD")
				}
				os.Exit(1)
			}
		} else {
			// If --all not passed, pass argument list to deletion logic
			_, errs, err := smdClient.DeleteRedfishEndpoints(token, xnameSlice...)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to delete redfish endpoints in SMD")
				os.Exit(1)
			}
			// Since smdClient.DeleteRedfishEndpoints does the deletion iteratively, we need to deal with
			// each error that might have occurred.
			var errorsOccurred = false
			for _, e := range errs {
				if err != nil {
					if errors.Is(e, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(e).Msg("SMD redfish endpoint deletion yielded unsuccessful HTTP response")
					} else {
						log.Logger.Error().Err(e).Msg("failed to delete redfish endpoint")
					}
					errorsOccurred = true
				}
			}
			// Warn the user if any errors occurred during deletion iterations
			if errorsOccurred {
				log.Logger.Warn().Msg("SMD redfish endpoint deletion completed with errors")
				os.Exit(1)
			}
		}
	},
}

func init() {
	rfeDeleteCmd.Flags().BoolP("all", "a", false, "delete all redfish endpoints in SMD")
	rfeDeleteCmd.Flags().StringP("payload", "f", "", "file containing the request payload; JSON format unless --payload-format specified")
	rfeDeleteCmd.Flags().String("payload-format", defaultPayloadFormat, "format of payload file (yaml,json) passed with --payload")
	rfeDeleteCmd.Flags().Bool("force", false, "do not ask before attempting deletion")

	rfeCmd.AddCommand(rfeDeleteCmd)
}
