// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package history

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	bss_lib "github.com/openchami/ochami/internal/cli/bss"
)

func NewCmd() *cobra.Command {
	// historyCmd represents the "bss history" command
	var historyCmd = &cobra.Command{
		Use:   "history",
		Args:  cobra.NoArgs,
		Short: "Fetch the endpoint history of BSS",
		Long: `Fetch the endpoint history of BSS.

See ochami-bss(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Create client to use for requests with runtime
			bssClient, err := bss_lib.GetClientWithRuntime(cmd, rt)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := rt.HandleToken(cmd); err != nil {
				return err
			}

			// If no ID flags are specified, get all boot parameters
			qstr := ""
			if cmd.Flag("xname").Changed || cmd.Flag("endpoint").Changed {
				values := url.Values{}
				if cmd.Flag("xname").Changed {
					x, err := cmd.Flags().GetString("xname")
					if err != nil {
						return cli.Errorf(cli.CodeUsage, "unable to fetch xname: %w", err)
					}
					values.Add("name", x)
				}
				if cmd.Flag("endpoint").Changed {
					e, err := cmd.Flags().GetString("endpoint")
					if err != nil {
						return cli.Errorf(cli.CodeUsage, "unable to fetch endpoint: %w", err)
					}
					values.Add("endpoint", e)
				}
				qstr = values.Encode()
			}

			// Send request
			httpEnv, err := bssClient.GetEndpointHistory(cmd.Context(), qstr)
			if err != nil {
				return cli.ClassifyClientError(err, "BSS endpoint history request yielded unsuccessful HTTP response", "failed to request endpoint history from BSS")
			}

			// Print output
			outBytes, err := client.FormatBody(httpEnv.Body, rt.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
			}
			if err := cli.WriteOutput(rt.Ios.Out(), outBytes); err != nil {
				return err
			}

			return nil
		},
	}

	// Create flags
	historyCmd.Flags().String("xname", "", "filter by xname")
	historyCmd.Flags().String("endpoint", "", "filter by endpoint")

	cli.AddFormatOutputFlag(historyCmd)
	historyCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return historyCmd
}
