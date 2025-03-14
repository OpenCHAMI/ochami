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

// groupDeleteCmd represents the smd-group-delete command
var groupDeleteCmd = &cobra.Command{
	Use:   "delete -f <payload_file> | <group_label>...",
	Short: "Delete one or more groups",
	Long: `Delete one or more groups. These can be specified by one or more group labels.
Alternatively, pass the group labels in an array of group structures
within a payload file and specify that file via -f. If - is used as
the argument to -f, the data is read from standard input.

This command sends a DELETE to SMD. An access token is required.`,
	Example: `  ochami smd group delete compute
  ochami smd group delete -f payload.json
  ochami smd group delete -f payload.yaml --payload-format yaml
  echo '<json_data>' | ochami smd group delete -f -
  echo '<yaml_data>' | ochami smd group delete -f - --payload-format yaml`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// With options, only one of:
		// - A payload file with -f
		// - A set of one or more group labels
		// must be passed.
		if len(args) == 0 {
			if !cmd.Flag("payload").Changed {
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
			os.Exit(1)
		}

		// This endpoint requires authentication, so a token is needed
		setTokenFromEnvVar(cmd)
		checkToken(cmd)

		// Create client to make request to SMD
		smdClient, err := smd.NewClient(smdBaseURI, insecure)
		if err != nil {
			log.Logger.Error().Err(err).Msg("error creating new SMD client")
			os.Exit(1)
		}

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(smdClient.OchamiClient)

		// Ask before attempting deletion unless --force was passed
		if !cmd.Flag("force").Changed {
			log.Logger.Debug().Msg("--force not passed, prompting user to confirm deletion")
			respDelete := loopYesNo("Really delete?")
			if !respDelete {
				log.Logger.Info().Msg("User aborted group deletion")
				os.Exit(0)
			} else {
				log.Logger.Debug().Msg("User answered affirmatively to delete groups")
			}
		}

		// Create list of group labels to delete
		var groups []smd.Group
		var gLabelSlice []string
		if cmd.Flag("payload").Changed {
			// Use payload file if passed
			handlePayload(cmd, &groups)
		} else {
			// ...otherwise, use passed CLI arguments
			gLabelSlice = args
		}

		// Perform deletion
		_, errs, err := smdClient.DeleteGroups(token, gLabelSlice...)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to delete groups in SMD")
			os.Exit(1)
		}
		// Since smdClient.DeleteGroups does the deletion iteratively, we need to deal with
		// each error that might have occurred.
		var errorsOccurred = false
		for _, e := range errs {
			if err != nil {
				if errors.Is(e, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(e).Msg("SMD group deletion yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(e).Msg("failed to delete group")
				}
				errorsOccurred = true
			}
		}
		// Warn the user if any errors occurred during deletion iterations
		if errorsOccurred {
			log.Logger.Warn().Msg("SMD group deletion completed with errors")
			os.Exit(1)
		}
	},
}

func init() {
	groupDeleteCmd.Flags().StringP("payload", "f", "", "file containing the request payload; JSON format unless --payload-format specified")
	groupDeleteCmd.Flags().StringP("payload-format", "F", defaultPayloadFormat, "format of payload file (yaml,json) passed with --payload")
	groupDeleteCmd.Flags().Bool("force", false, "do not ask before attempting deletion")

	groupCmd.AddCommand(groupDeleteCmd)
}
