// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
)

// bssDumpStateCmd represents the "bss dumpstate" command
var bssDumpStateCmd = &cobra.Command{
	Use:   "dumpstate",
	Args:  cobra.NoArgs,
	Short: "Retrieve the current state of BSS",
	Long: `Retrieve the current state of BSS.

See ochami-bss(1) for more details.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create client to use for requests
		bssClient := bssGetClient(cmd, false)

		// Send request
		httpEnv, err := bssClient.GetDumpState()
		if err != nil {
			if errors.Is(err, client.UnsuccessfulHTTPError) {
				log.Logger.Error().Err(err).Msg("BSS dump state request yielded unsuccessful HTTP response")
			} else {
				log.Logger.Error().Err(err).Msg("failed to request dump state from BSS")
			}
			logHelpError(cmd)
			os.Exit(1)
		}

		// Print output
		if outBytes, err := client.FormatBody(httpEnv.Body, formatOutput); err != nil {
			log.Logger.Error().Err(err).Msg("failed to format output")
			logHelpError(cmd)
			os.Exit(1)
		} else {
			fmt.Print(string(outBytes))
		}
	},
}

func init() {
	bssDumpStateCmd.Flags().VarP(&formatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	bssDumpStateCmd.RegisterFlagCompletionFunc("format-output", completionFormatData)

	bssCmd.AddCommand(bssDumpStateCmd)
}
