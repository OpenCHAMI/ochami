// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	boot_service_lib "github.com/openchami/ochami/internal/cli/boot_service"
)

func newCmdBootConfigList() *cobra.Command {
	// bootConfigListCmd represents the "boot config list" command
	var bootConfigListCmd = &cobra.Command{
		Use:   "list",
		Args:  cobra.NoArgs,
		Short: "List boot configurations",
		Long: `List boot configurations.

See ochami-boot(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			bootServiceClient, err := boot_service_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// Make request
			outBytes, err := bootServiceClient.ListBootConfigs(cli.Token, cli.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to list boot configurations: %w", err)
			}

			// Print output
			fmt.Print(string(outBytes))

			return nil
		},
	}

	// Create flags
	bootConfigListCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	bootConfigListCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return bootConfigListCmd
}
