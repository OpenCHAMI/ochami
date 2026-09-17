// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package service

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/cli/rcs"
	"github.com/openchami/ochami/pkg/format"
)

func newStatusCmd() *cobra.Command {
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Returns the status of the console service",
		Long: `Returns the status of the console service.

See ochami-rcs(1) for more details.`,
		Example: `  # Get console service status
  ochami rcs service status`,
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

			status, err := rcsClient.GetStatus(cmd.Context(), rt.Token)
			if err != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to get console service status: %w", err)
			}

			outBytes, err := format.MarshalData(status, rt.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
			}
			if err := cli.WriteString(rt.Ios.Out(), string(outBytes)+"\n"); err != nil {
				return err
			}

			return nil
		},
	}

	cli.AddFormatOutputFlag(statusCmd)
	statusCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return statusCmd
}
