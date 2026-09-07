// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package service

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	boot_service_lib "github.com/openchami/ochami/internal/cli/boot_service"
	"github.com/openchami/ochami/pkg/client"
)

func newCmdServiceStatus() *cobra.Command {
	// serviceStatusCmd represents the "boot service status" command
	var serviceStatusCmd = &cobra.Command{
		Use:   "status",
		Args:  cobra.NoArgs,
		Short: "Display status of the boot service",
		Long: `Display status of the boot service.

See ochami-boot(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			bootServiceClient, err := boot_service_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Make request
			outbytes, err := bootServiceClient.GetHealth(cli.FormatOutput)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					return cli.Errorf(cli.CodeHTTP, "failed to get boot-service health: %w", err)
				}
				return cli.Errorf(cli.CodeNetwork, "failed to get boot-service health: %w", err)
			}

			// Print output
			fmt.Fprint(cli.Ios.Out(), string(outbytes))

			return nil
		},
	}

	// Create flags
	serviceStatusCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	serviceStatusCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return serviceStatusCmd
}
