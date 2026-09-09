// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package bmc

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	boot_service_lib "github.com/openchami/ochami/internal/cli/boot_service"
)

func newCmdBootBmcList() *cobra.Command {
	// bootBmcListCmd represents the "boot bmc list" command
	var bootBmcListCmd = &cobra.Command{
		Use:   "list",
		Args:  cobra.NoArgs,
		Short: "List BMCs",
		Long: `List BMCs that boot-service knows about.

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
			outBytes, err := bootServiceClient.ListBMCs(cmd.Context(), cli.Token, cli.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to list BMCs: %w", err)
			}

			// Print output
			fmt.Fprint(cli.Ios.Out(), string(outBytes))

			return nil
		},
	}

	// Create flags
	bootBmcListCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	bootBmcListCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return bootBmcListCmd
}
