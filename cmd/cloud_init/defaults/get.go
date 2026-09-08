// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package defaults

import (
	"fmt"

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
			// Create client to use for requests
			cloudInitClient, err := cloud_init_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// Get data
			henv, err := cloudInitClient.GetDefaults(cli.Token)
			if err != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to get defaults: %w", err)
			}

			// Print in desired format
			outBytes, err := client.FormatBody(henv.Body, cli.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
			}
			fmt.Fprint(cli.Ios.Out(), string(outBytes))

			return nil
		},
	}

	// Create flags
	defaultsGetCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output")

	defaultsGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return defaultsGetCmd
}
