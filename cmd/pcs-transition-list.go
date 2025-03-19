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

// pcsTransitionListCmd represents the "pcs transition list" command
var pcsTransitionListCmd = &cobra.Command{
	Use:   "list",
	Args:  cobra.NoArgs,
	Short: "List active PCS transitions",
	Long: `List active PCS transitions.

See ochami-pcs(1) for more details.`,
	Example: `  # List transitions
  ochami pcs transition list`,
	Run: func(cmd *cobra.Command, args []string) {
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

		// Get transitions
		transitionsHttpEnv, err := pcsClient.GetTransitions(token)
		if err != nil {
			if errors.Is(err, client.UnsuccessfulHTTPError) {
				log.Logger.Fatal().Err(err).Msg("PCS transitions request yielded unsuccessful HTTP response")
			} else {
				log.Logger.Fatal().Err(err).Msg("failed to list PCS transitions")
			}
		}

		var output interface{}
		err = json.Unmarshal(transitionsHttpEnv.Body, &output)
		if err != nil {
			log.Logger.Fatal().Msg("failed to unmarshal transitions")
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
	pcsTransitionListCmd.Flags().StringP("format-output", "F", defaultOutputFormat, "format of output printed to standard output (json,json-pretty,yaml)")
	pcsTransitionCmd.AddCommand(pcsTransitionListCmd)
}
