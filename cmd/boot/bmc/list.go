// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package bmc

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	boot_service_lib "github.com/OpenCHAMI/ochami/internal/cli/boot_service"
	"github.com/OpenCHAMI/ochami/internal/log"
)

func newCmdBootBmcList() *cobra.Command {
	// bootBmcListCmd represents the "boot bmc list" command
	var bootBmcListCmd = &cobra.Command{
		Use:   "list",
		Args:  cobra.NoArgs,
		Short: "List BMCs",
		Long: `List BMCs that boot-service knows about.

See ochami-boot(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			bootServiceClient := boot_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Make request
			outBytes, err := bootServiceClient.ListBMCs(cli.Token, cli.FormatOutput)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to list BMCs")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Print output
			fmt.Print(string(outBytes))
		},
	}

	// Create flags
	bootBmcListCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	bootBmcListCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return bootBmcListCmd
}
