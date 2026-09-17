// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package service

import (
	"fmt"

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
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			rcsClient, err := rcs.GetClient(cmd)
			if err != nil {
				return err
			}

			status, err := rcsClient.GetStatus(cli.Token)
			if err != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to get console service status: %w", err)
			}

			outBytes, err := format.MarshalData(status, cli.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
			}
			fmt.Fprintln(cli.Ios.Out(), string(outBytes))

			return nil
		},
	}

	statusCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")
	statusCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return statusCmd
}
