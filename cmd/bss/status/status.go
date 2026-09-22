// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package status

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	bss_lib "github.com/openchami/ochami/internal/cli/bss"
)

func NewCmd() *cobra.Command {
	// statusCmd represents the "bss status" command
	var statusCmd = &cobra.Command{
		Deprecated: "use 'bss service status' instead. This command will be removed soon.",
		Use:        "status",
		Args:       cobra.NoArgs,
		Short:      "Get status of the Boot Script Service (BSS)",
		Long: `Get status of the Boot Script Service (BSS).

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
			} else if cmd.Flag("version").Changed {
				httpEnv, err = bssClient.GetStatus("version")
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
	statusCmd.Flags().Bool("all", false, "print all status data from BSS")
	statusCmd.Flags().Bool("storage", false, "print status of storage backend from BSS")
	statusCmd.Flags().Bool("smd", false, "print status of BSS connection to SMD")
	statusCmd.Flags().Bool("version", false, "print version of BSS")
	statusCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	statusCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)
	statusCmd.MarkFlagsMutuallyExclusive("all", "storage", "smd", "version")

	return statusCmd
}
