// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package service

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	smd_lib "github.com/openchami/ochami/internal/cli/smd"
)

func newCmdServiceStatus() *cobra.Command {
	// serviceStatusCmd represents the "smd service status" command
	var serviceStatusCmd = &cobra.Command{
		Use:   "status",
		Args:  cobra.NoArgs,
		Short: "Display status of the State Management Database (SMD)",
		Long: `Display status of the State Management Database (SMD).

See ochami-smd(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			smdClient, err := smd_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Determine which component to get status for and send request
			var httpEnv client.HTTPEnvelope
			if cmd.Flag("all").Changed {
				httpEnv, err = smdClient.GetStatus("all")
			} else {
				httpEnv, err = smdClient.GetStatus("")
			}
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					return cli.Errorf(cli.CodeHTTP, "SMD status request yielded unsuccessful HTTP response: %w", err)
				}
				return cli.Errorf(cli.CodeNetwork, "failed to get SMD status: %w", err)
			}

			// Print output
			outBytes, err := client.FormatBody(httpEnv.Body, cli.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
			}
			fmt.Fprint(cli.Ios.Out(), string(outBytes))

			return nil
		},
	}

	// Create flags
	serviceStatusCmd.Flags().Bool("all", false, "print all status data from SMD")
	serviceStatusCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	serviceStatusCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return serviceStatusCmd
}
