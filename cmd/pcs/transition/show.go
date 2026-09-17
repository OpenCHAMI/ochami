// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package transition

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/format"

	pcs_lib "github.com/openchami/ochami/internal/cli/pcs"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			transitionID := args[0]

			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Create client to use for requests with runtime
			pcsClient, err := pcs_lib.GetClientWithRuntime(cmd, rt)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := rt.HandleToken(cmd); err != nil {
				return err
			}

			// Get transition
			transitionHttpEnv, err := pcsClient.GetTransition(cmd.Context(), transitionID, rt.Token)
			if err != nil {
				return cli.ClassifyClientError(err, "PCS transitions request yielded unsuccessful HTTP response", "failed to get PCS transition")
			}

			// Unmarshal output
			var output interface{}
			err = json.Unmarshal(transitionHttpEnv.Body, &output)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to unmarshal transitions: %w", err)
			}

			// Print output
			outBytes, err := format.MarshalData(output, rt.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
			}
			if err := cli.WriteString(rt.Ios.Out(), string(outBytes)+"\n"); err != nil {
				return err
			}

			return nil
		},
	}

	// Format flags are inherited from root command
	cli.AddFormatOutputFlag(transitionShowCmd)
	transitionShowCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return transitionShowCmd
}
