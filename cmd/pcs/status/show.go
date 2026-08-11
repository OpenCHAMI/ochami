// SPDX-FileCopyrightText: © 2024-2026 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package status

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	pcs_lib "github.com/openchami/ochami/internal/cli/pcs"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/format"
)

type statusResponse struct {
	Status []map[string]interface{} `json:"status"`
}

func newCmdStatusShow() *cobra.Command {

	// pcsStatusShowCmd represents the "pcs status show" command
	var pcsStatusShowCmd = &cobra.Command{
		Use:   "show <xname>",
		Args:  cobra.ExactArgs(1),
		Short: "Show power status of target component",
		Long: `Show power status of target component.

See ochami-pcs(1) for more details.`,
		Example: `  # show power status of component
  ochami pcs status show x3000c0s15b0`,
		Run: func(cmd *cobra.Command, args []string) {
			xname := args[0]

			// Create client to use for requests
			pcsClient := pcs_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Get status
			statusHttpEnv, err := pcsClient.GetStatus([]string{xname}, "", "", cli.Token)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("PCS status request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to get power status")
				}
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			var output statusResponse

			err = json.Unmarshal(statusHttpEnv.Body, &output)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to unmarshal status")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Check if status array is empty
			if len(output.Status) == 0 {
				log.Logger.Error().Msg("no status found for the specified component")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Print output just for first element in status array
			if outBytes, err := format.MarshalData(output.Status[0], cli.FormatOutput); err != nil {
				log.Logger.Error().Err(err).Msg("failed to format output")
				cli.LogHelpError(cmd)
				os.Exit(1)
			} else {
				fmt.Println(string(outBytes))
			}
		},
	}

	// Define flags
	pcsStatusShowCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	pcsStatusShowCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return pcsStatusShowCmd
}
