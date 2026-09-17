// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package status

import (
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
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

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
			if err := cli.WriteOutput(rt.Ios.Out(), outBytes); err != nil {
				return err
			}

			return nil
		},
	}

	// Create flags
	statusCmd.Flags().Bool("all", false, "print all status data from SMD")

	cli.AddFormatOutputFlag(statusCmd)
	statusCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return statusCmd
}
