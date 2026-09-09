// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package node

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	boot_service_lib "github.com/openchami/ochami/internal/cli/boot_service"
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
			outBytes, err := bootServiceClient.GetNode(cmd.Context(), cli.Token, cli.FormatOutput, uid)
			if err != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to get node for %s: %w", uid, err)
			}

			// Print output
			fmt.Fprint(cli.Ios.Out(), string(outBytes))

			return nil
		},
	}

	// Create flags
	bootNodeGetCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	bootNodeGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return bootNodeGetCmd
}
