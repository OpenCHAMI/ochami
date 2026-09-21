// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package compep

import (
	"encoding/json"
	"errors"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	smd_lib "github.com/openchami/ochami/internal/cli/smd"
)

func newCmdCompepGet() *cobra.Command {
	// compepGetCmd represents the "smd compep get" command
	var compepGetCmd = &cobra.Command{
		Use:     "get [<xname>...]",
		Aliases: []string{"list"},
		Short:   "Get all component endpoints or a subset, identified by xname",
		Long: `Get all component endpoints or a subset, identified by xname.

See ochami-smd(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Create client to use for requests with runtime
			smdClient, err := smd_lib.GetClientWithRuntime(cmd, rt)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := rt.HandleToken(cmd); err != nil {
				return err
			}

			var httpEnv client.HTTPEnvelope
			if len(args) == 0 {
				// Get all ComponentEndpoints if no args passed
				httpEnv, err = smdClient.GetComponentEndpointsAll(cmd.Context(), rt.Token)
				if err != nil {
					return cli.ClassifyClientError(err, "SMD component endpoint request yielded unsuccessful HTTP response", "failed to request component endpoints from SMD")
				}

				// Print output
				outBytes, err := client.FormatBody(httpEnv.Body, rt.FormatOutput)
				if err != nil {
					return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
				}
				if err := cli.WriteOutput(rt.Ios.Out(), outBytes); err != nil {
					return err
				}
			} else {
				results := smdClient.GetComponentEndpoints(cmd.Context(), rt.Token, args...)
				// Since smdClient.GetComponentEndpoints does the fetching iteratively, we need to
				// deal with each error that might have occurred.
				var errorsOccurred = false
				for _, e := range results.Errors() {
					if e != nil {
						if errors.Is(e, client.UnsuccessfulHTTPError) {
							rt.Logger.Error().Err(e).Msg("SMD component endpoint request yielded unsuccessful HTTP response")
						} else {
							rt.Logger.Error().Err(e).Msg("failed to get component endpoint")
						}
						errorsOccurred = true
					}
				}

				// Put selected ComponentEndpoints into array and marshal
				type compEp struct {
					ComponentEndpoints []interface{} `json:"ComponentEndpoints" yaml:"ComponentEndpoints"`
				}
				var ceArr []interface{}
				for _, result := range results {
					if result.Err == nil {
						var ce interface{}
						err := json.Unmarshal(result.Value.Body, &ce)
						if err != nil {
							rt.Logger.Warn().Err(err).Msg("failed to unmarshal component endpoint")
							continue
						}
						ceArr = append(ceArr, ce)
					}
				}

				// Warn the user if any errors occurred during fetch iterations
				if errorsOccurred {
					return cli.Errorf(cli.CodeHTTP, "SMD component endpoint request completed with errors")
				}

				ces := compEp{ComponentEndpoints: ceArr}
				cesBytes, err := json.Marshal(ces)
				if err != nil {
					return cli.Errorf(cli.CodePayload, "failed to marshal list of component endpoints: %w", err)
				}

				// Print output
				outBytes, err := client.FormatBody(cesBytes, rt.FormatOutput)
				if err != nil {
					return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
				}
				if err := cli.WriteOutput(rt.Ios.Out(), outBytes); err != nil {
					return err
				}
			}

			return nil
		},
	}

	// Create flags

	cli.AddFormatOutputFlag(compepGetCmd)
	compepGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return compepGetCmd
}
