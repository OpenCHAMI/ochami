// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package transition

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/format"

	pcs_lib "github.com/OpenCHAMI/ochami/internal/cli/pcs"
)

func newCmdTransitionAbort() *cobra.Command {
	// transitionAbortCmd represents the "pcs transition abort" command
	var transitionAbortCmd = &cobra.Command{
		Use:   "abort <transition_id>",
		Args:  cobra.ExactArgs(1),
		Short: "Abort a PCS transition",
		Long: `Abort a PCS transition.

See ochami-pcs(1) for more details.`,
		Example: `  # Abort a transition
  ochami pcs transition abort 8f252166-c53c-435e-8354-e69649537a0f`,
		Run: func(cmd *cobra.Command, args []string) {
			transitionID := args[0]

			// Create client to use for requests
			pcsClient := pcs_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Abort the transition
			transitionHttpEnv, err := pcsClient.DeleteTransition(transitionID, cli.Token)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("PCS transition abort request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to abort PCS transition")
				}
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to abort transition")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			var output interface{}
			err = json.Unmarshal(transitionHttpEnv.Body, &output)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to unmarshal abort transitions response")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Print output
			if outBytes, err := format.MarshalData(output, cli.FormatOutput); err != nil {
				log.Logger.Error().Err(err).Msg("failed to format output")
				cli.LogHelpError(cmd)
				os.Exit(1)
			} else {
				fmt.Println(string(outBytes))
			}
		},
	}

	// Create flags
	transitionAbortCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	transitionAbortCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return transitionAbortCmd
}
