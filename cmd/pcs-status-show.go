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
	"github.com/OpenCHAMI/ochami/pkg/format"
)

type statusResponse struct {
	Status []map[string]interface{} `json:"status"`
}

// pcsStatusShowCmd represents the "pcs status show" command
var pcsStatusShowCmd = &cobra.Command{
	Use:   "show <xname>",
	Args:  cobra.ExactArgs(1),
	Short: "Show power status of target component",
	Long: `Show power status of target component .

See ochami-pcs(1) for more details.`,
	Example: `  # show power status of component
  ochami pcs status show x3000c0s15b0`,
	Run: func(cmd *cobra.Command, args []string) {
		xname := args[0]

		// Create client to use for requests
		pcsClient := pcsGetClient(cmd)

		// Handle token for this command
		handleToken(cmd)

		// Get status
		statusHttpEnv, err := pcsClient.GetStatus([]string{xname}, "", "", token)
		if err != nil {
			if errors.Is(err, client.UnsuccessfulHTTPError) {
				log.Logger.Error().Err(err).Msg("PCS status request yielded unsuccessful HTTP response")
			} else {
				log.Logger.Error().Err(err).Msg("failed to get power status")
			}
			logHelpError(cmd)
			os.Exit(1)
		}

		var output statusResponse

		err = json.Unmarshal(statusHttpEnv.Body, &output)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to unmarshal status")
			logHelpError(cmd)
			os.Exit(1)
		}

		// Check if status array is empty
		if len(output.Status) == 0 {
			log.Logger.Error().Msg("no status found for the specified component")
			logHelpError(cmd)
			os.Exit(1)
		}

		// Print output just for first element in status array
		if outBytes, err := format.MarshalData(output.Status[0], formatOutput); err != nil {
			log.Logger.Error().Err(err).Msg("failed to format output")
			logHelpError(cmd)
			os.Exit(1)
		} else {
			fmt.Println(string(outBytes))
		}
	},
}

func init() {
	pcsStatusShowCmd.Flags().VarP(&formatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	pcsStatusShowCmd.RegisterFlagCompletionFunc("format-output", completionFormatData)

	pcsStatusCmd.AddCommand(pcsStatusShowCmd)
}
