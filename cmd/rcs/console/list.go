// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package console

import (
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
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := rt.HandleToken(cmd); err != nil {
				return err
			}

			rcsClient, err := rcs.GetClientWithRuntime(cmd, rt)
			if err != nil {
				return err
			}
			consoles, err := rcsClient.ListConsoles(cmd.Context(), rt.Token)
			if err != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to list consoles: %w", err)
			}
			outBytes, err := format.MarshalData(consoles, rt.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
			}
			if err := cli.WriteString(rt.Ios.Out(), string(outBytes)+"\n"); err != nil {
				return err
			}

			return nil
		},
	}

	cli.AddFormatOutputFlag(listCmd)
	listCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return listCmd
}
