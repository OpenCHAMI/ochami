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
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			pcsClient := pcs_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Get transitions
			transitionsHttpEnv, err := pcsClient.GetTransitions(cli.Token)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("PCS transitions request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to list PCS transitions")
				}
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			var output interface{}
			err = json.Unmarshal(transitionsHttpEnv.Body, &output)
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
	transitionListCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	transitionListCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return transitionListCmd
}
