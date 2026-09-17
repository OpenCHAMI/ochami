// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package dumpstate

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	bss_lib "github.com/openchami/ochami/internal/cli/bss"
)

func NewCmd() *cobra.Command {
	// dumpstateCmd represents the "bss dumpstate" command
	var dumpstateCmd = &cobra.Command{
		Use:   "dumpstate",
		Args:  cobra.NoArgs,
		Short: "Retrieve the current state of BSS",
		Long: `Retrieve the current state of BSS.

See ochami-bss(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			bssClient, err := bss_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Send request
			httpEnv, err := bssClient.GetDumpstate()
			if err != nil {
				return cli.ClassifyClientError(err, "BSS dump state request yielded unsuccessful HTTP response", "failed to request dump state from BSS")

			}

			// Print output
			outBytes, err := client.FormatBody(httpEnv.Body, cli.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
			}
			fmt.Fprint(cli.Ios.Out(), string(outBytes))

			return nil
		},
	}

	// Create flags
	dumpstateCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	dumpstateCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return dumpstateCmd
}
