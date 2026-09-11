// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package status

import (
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
			// Try to get runtime from context (new approach)
			if rt, ok := cli.FromContext(cmd.Context()); ok {
				// Create client to use for requests with runtime
				smdClient, err := smd_lib.GetClientWithRuntime(cmd, rt)
				if err != nil {
					return err
				}

				// Handle token for this command
				if err := rt.HandleToken(cmd); err != nil {
					return err
				}

				// Determine which component to get status for and send request
				var httpEnv client.HTTPEnvelope
				if cmd.Flag("all").Changed {
					httpEnv, err = smdClient.GetStatus(cmd.Context(), "all")
				} else {
					httpEnv, err = smdClient.GetStatus(cmd.Context(), "")
				}
				if err != nil {
					return cli.ClassifyClientError(err, "SMD status request yielded unsuccessful HTTP response", "failed to get SMD status")
				}

				// Print output
				outBytes, err := client.FormatBody(httpEnv.Body, rt.FormatOutput)
				if err != nil {
					return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
				}
				fmt.Fprint(rt.Ios.Out(), string(outBytes))

				return nil
			}

			// Fallback to old approach during transition
			// Create client to use for requests
			smdClient, err := smd_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Determine which component to get status for and send request
			var httpEnv client.HTTPEnvelope
			if cmd.Flag("all").Changed {
				httpEnv, err = smdClient.GetStatus(cmd.Context(), "all")
			} else {
				httpEnv, err = smdClient.GetStatus(cmd.Context(), "")
			}
			if err != nil {
				return cli.ClassifyClientError(err, "SMD status request yielded unsuccessful HTTP response", "failed to get SMD status")

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
