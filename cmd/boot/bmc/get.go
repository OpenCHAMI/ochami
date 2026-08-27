// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package bmc

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	boot_service_lib "github.com/openchami/ochami/internal/cli/boot_service"
	"github.com/openchami/ochami/pkg/client"
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
   ochami boot bmc get bmc-773d99bf`,
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
			outBytes, err := bootServiceClient.GetBMC(cli.Token, cli.FormatOutput, uid)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					return cli.Errorf(cli.CodeHTTP, "failed to get BMC info for %s: %w", uid, err)
				}
				return cli.Errorf(cli.CodeNetwork, "failed to get BMC info for %s: %w", uid, err)
			}

			// Print output
			fmt.Print(string(outBytes))

			return nil
		},
	}

	// Create flags
	bootBmcGetCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	bootBmcGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return bootBmcGetCmd
}
