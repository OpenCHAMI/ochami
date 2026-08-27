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

func newCmdServiceVersion() *cobra.Command {
	// serviceVersionCmd represents the "bss service version" command
	var serviceVersionCmd = &cobra.Command{
		Use:   "version",
		Args:  cobra.NoArgs,
		Short: "Print version of the Boot Script Service (BSS)",
		Long: `Print version of the Boot Script Service (BSS).

See ochami-bss(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			bssClient, err := bss_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Determine which component to get status for and send request
			httpEnv, err := bssClient.GetStatus("version")
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					return cli.Errorf(cli.CodeHTTP, "BSS version request yielded unsuccessful HTTP response: %w", err)
				}
				return cli.Errorf(cli.CodeNetwork, "failed to get BSS version: %w", err)
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

	return serviceVersionCmd
}
