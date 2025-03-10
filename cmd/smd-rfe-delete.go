// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"os"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/smd"
	"github.com/spf13/cobra"
)

// rfeDeleteCmd represents the smd-rfe-delete command
var rfeDeleteCmd = &cobra.Command{
	Use:   "delete (-d (<payload_data> | @<payload_file>)) | --all | <xname>...",
	Short: "Delete one or more redfish endpoints",
	Long: `Delete one or more redfish endpoints. These can be specified
by one or more xnames. Alternatively, pass -d to pass raw
payload data or (if flag argument starts with @) a file
containing the payload data. -f can be specified to change
the format of the input payload data ('json' by default),
but the rules above still apply for the payload. If "-" is
used as the input payload filename, the data is read from
standard input.

This command sends a DELETE to SMD. An access token is required.

See ochami-smd(1) for more details.`,
	Example: `  # Delete a redfish endpoint using CLI flags
  ochami smd rfe delete x3000c1s7b56
  ochami smd rfe delete x3000c1s7b56 x3000c1s7b56
  ochami smd rfe delete --all

  # Delete redfish endpoints using input payload file
  ochami smd rfe delete -d @payload.json
  ochami smd rfe delete -d @payload.yaml -f yaml

  # Delete redfish endpoints using data from standard input
  echo '<json_data>' | ochami smd rfe delete -d @-
  echo '<yaml_data>' | ochami smd rfe delete -d @- -f yaml`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// With options, only one of:
		// - A payload file with -f
		// - --all
		// - A set of one or more xnames
		// must be passed.
		if len(args) == 0 {
			if !cmd.Flag("all").Changed && !cmd.Flag("data").Changed {
				printUsageHandleError(cmd)
				os.Exit(0)
			}
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Without a base URI, we cannot do anything
		smdBaseURI, err := getBaseURISMD(cmd)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for SMD")
			logHelpError(cmd)
			os.Exit(1)
		}

		// This endpoint requires authentication, so a token is needed
		setTokenFromEnvVar(cmd)
		checkToken(cmd)

		// Create client to make request to SMD
		smdClient, err := smd.NewClient(smdBaseURI, insecure)
		if err != nil {
			log.Logger.Error().Err(err).Msg("error creating new SMD client")
			logHelpError(cmd)
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
		var rfeSlice smd.RedfishEndpointSlice
		var xnameSlice []string
		if cmd.Flag("data").Changed {
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
				logHelpError(cmd)
				os.Exit(1)
			}
		} else {
			// If --all not passed, pass argument list to deletion logic
			_, errs, err := smdClient.DeleteRedfishEndpoints(token, xnameSlice...)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to delete redfish endpoints in SMD")
				logHelpError(cmd)
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
				logHelpError(cmd)
			}
		}
	},
}

func init() {
	rfeDeleteCmd.Flags().BoolP("all", "a", false, "delete all redfish endpoints in SMD")
	rfeDeleteCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	rfeDeleteCmd.Flags().StringP("format-input", "f", defaultInputFormat, "format of input payload data (json,yaml)")
	rfeDeleteCmd.Flags().Bool("force", false, "do not ask before attempting deletion")

	rfeCmd.AddCommand(rfeDeleteCmd)
}
