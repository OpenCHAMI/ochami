// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package service

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/cli/rcs"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/format"
)

func newStatusCmd() *cobra.Command {
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Returns the status of the console service",
		Long: `Returns the status of the console service.

See ochami-rcs(1) for more details.`,
		Example: `  # Get console service status
  ochami rcs service status`,
		Run: func(cmd *cobra.Command, args []string) {
			cli.HandleToken(cmd)

			rcsClient := rcs.GetClient(cmd)

			status, err := rcsClient.GetStatus(cli.Token)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to get console service status")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			if outBytes, err := format.MarshalData(status, cli.FormatOutput); err != nil {
				log.Logger.Error().Err(err).Msg("failed to format output")
				cli.LogHelpError(cmd)
				os.Exit(1)
			} else {
				fmt.Println(string(outBytes))
			}
		},
	}

	statusCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")
	statusCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return statusCmd
}
