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
			// Create client to use for requests
			pcsClient, err := pcs_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// Get transitions
			transitionsHttpEnv, err := pcsClient.GetTransitions(cli.Token)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					return cli.Errorf(cli.CodeHTTP, "PCS transitions request yielded unsuccessful HTTP response: %w", err)
				}
				return cli.Errorf(cli.CodeNetwork, "failed to list PCS transitions: %w", err)
			}

			var output interface{}
			err = json.Unmarshal(transitionsHttpEnv.Body, &output)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to unmarshal transitions: %w", err)
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
	transitionListCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	transitionListCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return transitionListCmd
}
