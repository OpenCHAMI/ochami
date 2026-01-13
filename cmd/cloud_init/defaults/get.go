// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package defaults

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"

	// Command library
	cloud_init_lib "github.com/OpenCHAMI/ochami/internal/cli/cloud_init"
)

func newCmdDefaultsGet() *cobra.Command {
	// defaultsGetCmd represents the "cloud-init defaults get" command
	var defaultsGetCmd = &cobra.Command{
		Use:   "get",
		Args:  cobra.NoArgs,
		Short: "Get cloud-init default meta-data for a cluster",
		Long: `Get cloud-init default meta-data for a cluster.

See ochami-cloud-init(1) for more details.`,
		Example: `  ochami cloud-init defaults get`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			cloudInitClient := cloud_init_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Get data
			henv, err := cloudInitClient.GetDefaults(cli.Token)
			if err != nil {
				log.Logger.Error().Err(err).Msgf("failed to get defaults")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Print in desired format
			if outBytes, err := client.FormatBody(henv.Body, cli.FormatOutput); err != nil {
				log.Logger.Error().Err(err).Msg("failed to format output")
				cli.LogHelpError(cmd)
				os.Exit(1)
			} else {
				fmt.Print(string(outBytes))
			}
		},
	}

	// Create flags
	defaultsGetCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output")

	defaultsGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return defaultsGetCmd
}
