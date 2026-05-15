// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package console

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/cli/rcs"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/format"
)

func newListCmd() *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Returns a list of the available consoles",
		Long: `Returns a list of the available consoles.

See ochami-rcs(1) for more details.`,
		Example: `  # List available consoles
  ochami rcs console list`,
		Run: func(cmd *cobra.Command, args []string) {
			cli.HandleToken(cmd)

			rcsClient := rcs.GetClient(cmd)
			consoles, err := rcsClient.ListConsoles(cli.Token)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to list consoles")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
			if outBytes, err := format.MarshalData(consoles, cli.FormatOutput); err != nil {
				log.Logger.Error().Err(err).Msg("failed to format output")
				cli.LogHelpError(cmd)
				os.Exit(1)
			} else {
				fmt.Println(string(outBytes))
			}
		},
	}

	listCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")
	listCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return listCmd
}
