// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package transition

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/format"

	pcs_lib "github.com/openchami/ochami/internal/cli/pcs"
)

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
		RunE: func(cmd *cobra.Command, args []string) error {
			operation := args[0]

			if !isValidOperation(operation) {
				// Include invalid operation in error message
				return cli.Errorf(cli.CodeUsage, "invalid operation: %s", operation)
			}

			// Create client to use for requests
			pcsClient, err := pcs_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// Get the list of target components
			xnames, _ := cmd.Flags().GetStringSlice("xname")

			// Create transition
			transitionHttpEnv, err := pcsClient.CreateTransition(operation, nil, xnames, cli.Token)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					return cli.Errorf(cli.CodeHTTP, "PCS transition create request yielded unsuccessful HTTP response: %w", err)
				}
				return cli.Errorf(cli.CodeNetwork, "failed to create transition: %w", err)
			}

			// Unmarshall the transition
			var output createOutput
			err = json.Unmarshal(transitionHttpEnv.Body, &output)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to unmarshal output: %w", err)
			}

			// Print output
			outBytes, err := format.MarshalData(output, cli.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
			}
			fmt.Println(string(outBytes))

			return nil
		},
	}

	// Create flags
	transitionStartCmd.Flags().StringSliceP("xname", "x", []string{}, "The list of target components")
	_ = transitionStartCmd.MarkFlagRequired("xname")

	transitionStartCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	transitionStartCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return transitionStartCmd
}
