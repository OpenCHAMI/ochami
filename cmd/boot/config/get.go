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

func newCmdBootConfigGet() *cobra.Command {
	// bootConfigGetCmd represents the "boot config get" command
	var bootConfigGetCmd = &cobra.Command{
		Use:   "get <uid>",
		Args:  cobra.ExactArgs(1),
		Short: "Get a boot configuration by its UID",
		Long: `Get a boot configuration by its UID.

See ochami-boot(1) for more details.`,
		Example: `  # Get boot configuration for node
  ochami boot config get boo-ebf2a27a`,
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

			uid := args[0]

			// Make request
			outBytes, err := bootServiceClient.GetBootConfig(cmd.Context(), cli.Token, cli.FormatOutput, uid)
			if err != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to get boot configuration for %s: %w", uid, err)
			}

			// Print output
			fmt.Fprint(cli.Ios.Out(), string(outBytes))

			return nil
		},
	}

	// Create flags
	bootConfigGetCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	bootConfigGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return bootConfigGetCmd
}
