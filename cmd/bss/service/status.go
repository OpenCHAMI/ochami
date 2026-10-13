// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package service

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	bss_lib "github.com/openchami/ochami/internal/cli/bss"
)

func newCmdServiceStatus() *cobra.Command {
	// serviceStatusCmd represents the "bss service status" command
	var serviceStatusCmd = &cobra.Command{
		Use:   "status",
		Args:  cobra.NoArgs,
		Short: "Display status of the Boot Script Service (BSS)",
		Long: `Display status of the Boot Script Service (BSS).

See ochami-bss(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			bssClient, err := bss_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Determine which component to get status for and send request
			var httpEnv client.HTTPEnvelope
			if cmd.Flag("all").Changed {
				httpEnv, err = bssClient.GetStatus("all")
			} else if cmd.Flag("storage").Changed {
				httpEnv, err = bssClient.GetStatus("storage")
			} else if cmd.Flag("smd").Changed {
				httpEnv, err = bssClient.GetStatus("smd")
			} else {
				httpEnv, err = bssClient.GetStatus("")
			}
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					return cli.Errorf(cli.CodeHTTP, "BSS status request yielded unsuccessful HTTP response: %w", err)
				}
				return cli.Errorf(cli.CodeNetwork, "failed to get BSS status: %w", err)
			}

			// Print output
			outBytes, err := client.FormatBody(httpEnv.Body, cli.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
			}
			fmt.Print(string(outBytes))

			return nil
		},
	}

	// Create flags
	serviceStatusCmd.Flags().Bool("all", false, "print all status data from BSS")
	serviceStatusCmd.Flags().Bool("storage", false, "print status of storage backend from BSS")
	serviceStatusCmd.Flags().Bool("smd", false, "print status of BSS connection to SMD")
	serviceStatusCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	serviceStatusCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)
	serviceStatusCmd.MarkFlagsMutuallyExclusive("all", "storage", "smd")

	return serviceStatusCmd
}
