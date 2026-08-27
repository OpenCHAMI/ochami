// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package console

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/cli/rcs"
	"github.com/openchami/ochami/pkg/format"
)

func newListCmd() *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Returns a list of the available consoles",
		Long: `Returns a list of the available consoles.

See ochami-rcs(1) for more details.`,
		Example: `  # List available consoles
  ochami rcs console list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			rcsClient, err := rcs.GetClient(cmd)
			if err != nil {
				return err
			}
			consoles, err := rcsClient.ListConsoles(cli.Token)
			if err != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to list consoles: %w", err)
			}
			outBytes, err := format.MarshalData(consoles, cli.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
			}
			fmt.Println(string(outBytes))

			return nil
		},
	}

	listCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")
	listCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return listCmd
}
