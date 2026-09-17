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

// bssStatusClient is the subset of the BSS client that the "bss service status"
// command depends on. Defining it here (rather than importing a concrete client
// type) lets tests substitute a fake and exercise the command's response
// handling and error mapping without a live service.
type bssStatusClient interface {
	GetStatus(component string) (client.HTTPEnvelope, error)
}

// bssStatusClientProvider builds a bssStatusClient from the command context. The
// production provider constructs a real BSS client; tests inject their own.
type bssStatusClientProvider func(cmd *cobra.Command) (bssStatusClient, error)

// realBSSStatusClient is the production provider used by newCmdServiceStatus. It
// delegates to the internal CLI helper that wires up a configured BSS client.
func realBSSStatusClient(cmd *cobra.Command) (bssStatusClient, error) {
	return bss_lib.GetClient(cmd)
}

func newCmdServiceStatus() *cobra.Command {
	return newCmdServiceStatusWithClient(realBSSStatusClient)
}

// newCmdServiceStatusWithClient builds the "bss service status" command using
// the given client provider. It exists so tests can inject a fake client.
func newCmdServiceStatusWithClient(getClient bssStatusClientProvider) *cobra.Command {
	// serviceStatusCmd represents the "bss service status" command
	var serviceStatusCmd = &cobra.Command{
		Use:   "status",
		Args:  cobra.NoArgs,
		Short: "Display status of the Boot Script Service (BSS)",
		Long: `Display status of the Boot Script Service (BSS).

See ochami-bss(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			bssClient, err := getClient(cmd)
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
