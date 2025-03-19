// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/pcs"
	"github.com/OpenCHAMI/ochami/pkg/format"
)

var xnames []string
var operation string

// validOperations returns a list of valid PCS operations
func validOperations() []string {
	return []string{"force-off", "hard-restart", "off", "on", "reinit", "soft-off", "soft-restart"}
}

// isValidOperation checks if the given operation is a valid PCS operation
func isValidOperation(operation string) bool {
	for _, op := range validOperations() {
		if operation == op {
			return true
		}
	}

	return false
}

// createOutput represents the output of the start transition command
type createOutput struct {
	TransitionID string
	Operation    string
}

// pcsTransitionStartCmd represents the "pcs transition start" command
var pcsTransitionStartCmd = &cobra.Command{
	Use:   "start",
	Args:  cobra.ExactArgs(1),
	Short: "Start a PCS transition",
	Long: `Start a PCS transition.

See ochami-pcs(1) for more details.`,
	Example: `  # Turn on a set of nodes
  ochami pcs transition start --xname "x0c0s7b0n1,x0c0s7b0n0,x0c0s4b0n1" on`,
	Run: func(cmd *cobra.Command, args []string) {
		operation = args[0]

		if !isValidOperation(operation) {
			// Include invalid operation in error message
			log.Logger.Error().Str("operation", operation).Msg("Invalid operation")
			logHelpError(cmd)
			os.Exit(1)
		}

		// Without a base URI, we cannot do anything
		pcsBaseURI, err := getBaseURIPCS(cmd)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for PCS")
			logHelpError(cmd)
			os.Exit(1)
		}

		// This endpoint requires authentication, so a token is needed
		setTokenFromEnvVar(cmd)
		checkToken(cmd)

		// Create client to make request to PCS
		pcsClient, err := pcs.NewClient(pcsBaseURI, insecure)
		if err != nil {
			log.Logger.Fatal().Err(err).Msg("error creating new PCS client")
		}

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(pcsClient.OchamiClient)

		// Get the list of target components
		xnames, err = cmd.Flags().GetStringSlice("xname")
		if err != nil {
			log.Logger.Fatal().Err(err).Msg("failed to get value for --xname")
		}

		// Create transition
		transitionHttpEnv, err := pcsClient.CreateTransition(operation, nil, xnames, token)
		if err != nil {
			if errors.Is(err, client.UnsuccessfulHTTPError) {
				log.Logger.Fatal().Err(err).Msg("PCS transition create request yielded unsuccessful HTTP response")
			} else {
				log.Logger.Fatal().Err(err).Msg("failed to create transition")
			}
		}

		// Unmarshall the transition
		var output createOutput
		err = json.Unmarshal(transitionHttpEnv.Body, &output)
		if err != nil {
			log.Logger.Fatal().Msg("failed to unmarshal output")
		}

		// Print output
		outFmt, err := cmd.Flags().GetString("format-output")
		if err != nil {
			log.Logger.Fatal().Err(err).Msg("failed to get value for --format-output")
		}
		if outBytes, err := format.FormatData(output, outFmt); err != nil {
			log.Logger.Fatal().Err(err).Msg("failed to format output")
		} else {
			fmt.Println(string(outBytes))
		}
	},
}

func init() {
	pcsTransitionStartCmd.Flags().StringSliceP("xname", "x", []string{}, "The list of target components")
	if err := pcsTransitionStartCmd.MarkFlagRequired("xname"); err != nil {
		log.Logger.Fatal().Err(err).Msg("failed to mark xname as required")
	}

	pcsTransitionStartCmd.Flags().StringP("format-output", "F", defaultOutputFormat, "format of output printed to standard output (json,json-pretty,yaml)")
	pcsTransitionCmd.AddCommand(pcsTransitionStartCmd)
}
