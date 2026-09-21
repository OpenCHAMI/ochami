// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package defaults

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	// Command library
	cloud_init_lib "github.com/openchami/ochami/internal/cli/cloud_init"
)

func newCmdDefaultsGet() *cobra.Command {
	// defaultsGetCmd represents the "cloud-init defaults get" command
	var defaultsGetCmd = &cobra.Command{
		Use:     "get",
		Aliases: []string{"list"},
		Args:    cobra.NoArgs,
		Short:   "Get cloud-init default meta-data for a cluster",
		Long: `Get cloud-init default meta-data for a cluster.

See ochami-cloud-init(1) for more details.`,
		Example: `  ochami cloud-init defaults get`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Create client to use for requests
			cloudInitClient, err := cloud_init_lib.GetClientWithRuntime(cmd, rt)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := rt.HandleToken(cmd); err != nil {
				return err
			}

			// Get data
			henv, err := cloudInitClient.GetDefaults(cmd.Context(), rt.Token)
			if err != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to get defaults: %w", err)
			}

			// Print in desired format
			outBytes, err := client.FormatBody(henv.Body, rt.FormatOutput)
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

	cli.AddFormatOutputFlag(defaultsGetCmd)
	defaultsGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return defaultsGetCmd
}
