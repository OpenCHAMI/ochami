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

func newCmdBootBmcGet() *cobra.Command {
	// bootBmcGetCmd represents the "boot bmc get" command
	var bootBmcGetCmd = &cobra.Command{
		Use:   "get <uid>",
		Args:  cobra.ExactArgs(1),
		Short: "Get a BMC by its UID",
		Long: `Get a BMC by its UID.

See ochami-boot(1) for more details.`,
		Example: `  # Get info about a BMC
  ochami boot node get bmc-773d99bf`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			bootServiceClient := boot_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			uid := args[0]

			// Make request
			outBytes, err := bootServiceClient.GetBMC(cli.Token, cli.FormatOutput, uid)
			if err != nil {
				log.Logger.Error().Err(err).Msgf("failed to get BMC info for %s", uid)
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Print output
			fmt.Print(string(outBytes))
		},
	}

	// Create flags
	bootBmcGetCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	bootBmcGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return bootBmcGetCmd
}
