// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package status

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	smd_lib "github.com/openchami/ochami/internal/cli/smd"
)

func NewCmd() *cobra.Command {
	// statusCmd represents the "smd status" command
	var statusCmd = &cobra.Command{
		Deprecated: "use 'smd service status' instead. This command will be removed soon.",
		Use:        "status",
		Args:       cobra.NoArgs,
		Short:      "Get status of the State Management Database (SMD)",
		Long: `Get status of the State Management Database (SMD).

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
	statusCmd.Flags().Bool("all", false, "print all status data from SMD")
	statusCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	statusCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return statusCmd
}
