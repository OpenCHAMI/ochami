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

func newCmdTransitionStart() *cobra.Command {
	// transitionStartCmd represents the "pcs transition start" command
	var transitionStartCmd = &cobra.Command{
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
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Create client to use for requests
			pcsClient := pcs_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Get the list of target components
			var err error
			xnames, err = cmd.Flags().GetStringSlice("xname")
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to get value for --xname")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Create transition
			transitionHttpEnv, err := pcsClient.CreateTransition(operation, nil, xnames, cli.Token)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("PCS transition create request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to create transition")
				}
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Unmarshall the transition
			var output createOutput
			err = json.Unmarshal(transitionHttpEnv.Body, &output)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to unmarshal output")
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
	transitionStartCmd.Flags().StringSliceP("xname", "x", []string{}, "The list of target components")
	if err := transitionStartCmd.MarkFlagRequired("xname"); err != nil {
		log.Logger.Fatal().Err(err).Msg("failed to mark xname as required")
	}

	transitionStartCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	transitionStartCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return transitionStartCmd
}
