// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package transition

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client/pcs"
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

// transitionStartOptions holds the flag values for the pcs transition start command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type transitionStartOptions struct {
	Xnames []string
}

// createOutput represents the output of the start transition command
type createOutput struct {
	TransitionID string
	Operation    string
}

// runCoreTransitionStartWithRuntime contains the core logic for the pcs transition start command using runtime.
// It takes the parsed options and performs the actual work of starting a transition.
func runCoreTransitionStartWithRuntime(cmd *cobra.Command, opts *transitionStartOptions, args []string, pcsClient *pcs.PCSClient, rt *cli.Runtime) error {
	operation := args[0]

	if !isValidOperation(operation) {
		// Include invalid operation in error message
		return cli.Errorf(cli.CodeUsage, "invalid operation: %s", operation)
	}

	// Handle token for this command
	if err := rt.HandleToken(cmd); err != nil {
		return err
	}

	// Create transition
	transitionHttpEnv, err := pcsClient.CreateTransition(cmd.Context(), operation, nil, opts.Xnames, rt.Token)
	if err != nil {
		return cli.ClassifyClientError(err, "PCS transition create request yielded unsuccessful HTTP response", "failed to create transition")
	}

	// Unmarshall the transition
	var output createOutput
	err = json.Unmarshal(transitionHttpEnv.Body, &output)
	if err != nil {
		return cli.Errorf(cli.CodePayload, "failed to unmarshal output: %w", err)
	}

	// Print output
	outBytes, err := format.MarshalData(output, rt.FormatOutput)
	if err != nil {
		return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
	}
	fmt.Fprintln(rt.Ios.Out(), string(outBytes))

	return nil
}

// runCoreTransitionStart contains the core logic for the pcs transition start command using global state.
// This is the fallback during migration.
func runCoreTransitionStart(cmd *cobra.Command, opts *transitionStartOptions, args []string, pcsClient *pcs.PCSClient) error {
	operation := args[0]

	if !isValidOperation(operation) {
		// Include invalid operation in error message
		return cli.Errorf(cli.CodeUsage, "invalid operation: %s", operation)
	}

	// Handle token for this command
	if err := cli.HandleToken(cmd); err != nil {
		return err
	}

	// Create transition
	transitionHttpEnv, err := pcsClient.CreateTransition(cmd.Context(), operation, nil, opts.Xnames, cli.Token)
	if err != nil {
		return cli.ClassifyClientError(err, "PCS transition create request yielded unsuccessful HTTP response", "failed to create transition")
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
	fmt.Fprintln(cli.Ios.Out(), string(outBytes))

	return nil
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
			// Try to get runtime from context first (new approach)
			if rt, ok := cli.FromContext(cmd.Context()); ok {
				// Create client to use for requests with runtime
				pcsClient, err := pcs_lib.GetClientWithRuntime(cmd, rt)
				if err != nil {
					return err
				}

				// Extract options from flags
				// Since flags are registered with the correct types on this command,
				// these Get* calls cannot fail, so we ignore errors with explicit comments
				opts := &transitionStartOptions{}
				if cmd.Flag("xname").Changed {
					opts.Xnames, _ = cmd.Flags().GetStringSlice("xname") //nolint:errcheck // Flag registered with matching type, error impossible
				}

				return runCoreTransitionStartWithRuntime(cmd, opts, args, pcsClient, rt)
			}

			// Fallback to old approach during transition
			// Create client to use for requests
			pcsClient, err := pcs_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Extract options from flags
			// Since flags are registered with the correct types on this command,
			// these Get* calls cannot fail, so we ignore errors with explicit comments
			opts := &transitionStartOptions{}
			if cmd.Flag("xname").Changed {
				opts.Xnames, _ = cmd.Flags().GetStringSlice("xname") //nolint:errcheck // Flag registered with matching type, error impossible
			}

			return runCoreTransitionStart(cmd, opts, args, pcsClient)
		},
	}

	// Create flags
	transitionStartCmd.Flags().StringSliceP("xname", "x", []string{}, "The list of target components")
	_ = transitionStartCmd.MarkFlagRequired("xname") //nolint:errcheck // Flag registered immediately above, error impossible

	transitionStartCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	transitionStartCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return transitionStartCmd
}
