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

func newCmdTransitionList() *cobra.Command {
	// transitionListCmd represents the "pcs transition list" command
	var transitionListCmd = &cobra.Command{
		Use:   "list",
		Args:  cobra.NoArgs,
		Short: "List active PCS transitions",
		Long: `List active PCS transitions.

See ochami-pcs(1) for more details.`,
		Example: `  # List transitions
  ochami pcs transition list`,
		RunE: func(cmd *cobra.Command, args []string) error {
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

			// Get transitions
			transitionsHttpEnv, err := pcsClient.GetTransitions(cmd.Context(), rt.Token)
			if err != nil {
				return cli.ClassifyClientError(err, "PCS transitions request yielded unsuccessful HTTP response", "failed to list PCS transitions")
			}

			var output interface{}
			err = json.Unmarshal(transitionsHttpEnv.Body, &output)
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
	cli.AddFormatOutputFlag(transitionListCmd)
	transitionListCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return transitionListCmd
}
