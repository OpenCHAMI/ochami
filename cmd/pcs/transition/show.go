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

func newCmdTransitionShow() *cobra.Command {
	// transition show Cmd represents the "pcs transition show" command
	var transitionShowCmd = &cobra.Command{
		Use:   "show <transition_id>",
		Args:  cobra.ExactArgs(1),
		Short: "Show details of a PCS transition",
		Long: `Show details of a PCS transition.

See ochami-pcs(1) for more details.`,
		Example: `  # Show a transition
  ochami pcs transition show 8f252166-c53c-435e-8354-e69649537a0f`,
		Run: func(cmd *cobra.Command, args []string) {
			transitionID := args[0]

			// Create client to use for requests
			pcsClient := pcs_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Get transition
			transitionHttpEnv, err := pcsClient.GetTransition(transitionID, cli.Token)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("PCS transitions request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to get PCS transition")
				}
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Unmarshal output
			var output interface{}
			err = json.Unmarshal(transitionHttpEnv.Body, &output)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to unmarshal transitions")
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
	transitionShowCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	transitionShowCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return transitionShowCmd
}
