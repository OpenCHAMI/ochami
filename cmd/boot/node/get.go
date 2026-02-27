// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package node

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	boot_service_lib "github.com/OpenCHAMI/ochami/internal/cli/boot_service"
	"github.com/OpenCHAMI/ochami/internal/log"
)

func newCmdBootNodeGet() *cobra.Command {
	// bootNodeGetCmd represents the "boot node get" command
	var bootNodeGetCmd = &cobra.Command{
		Use:   "get <uid>",
		Args:  cobra.ExactArgs(1),
		Short: "Get a node by its UID",
		Long: `Get a node by its UID.

See ochami-boot(1) for more details.`,
		Example: `  # Get info about a node
  ochami boot node get nod-bc76f7f2`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			bootServiceClient := boot_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			uid := args[0]

			// Make request
			outBytes, err := bootServiceClient.GetNode(cli.Token, cli.FormatOutput, uid)
			if err != nil {
				log.Logger.Error().Err(err).Msgf("failed to get boot configuration for %s", uid)
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Print output
			fmt.Print(string(outBytes))
		},
	}

	// Create flags
	bootNodeGetCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	bootNodeGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return bootNodeGetCmd
}
