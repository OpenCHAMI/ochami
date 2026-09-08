// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package compep

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
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
			// Create client to use for requests
			smdClient, err := smd_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			var httpEnv client.HTTPEnvelope
			if len(args) == 0 {
				// Get all ComponentEndpoints if no args passed
				httpEnv, err = smdClient.GetComponentEndpointsAll(cli.Token)
				if err != nil {
					return cli.ClassifyClientError(err, "SMD component endpoint request yielded unsuccessful HTTP response", "failed to request component endpoints from SMD")

				}

				// Print output
				outBytes, err := client.FormatBody(httpEnv.Body, cli.FormatOutput)
				if err != nil {
					return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
				}
				fmt.Fprint(cli.Ios.Out(), string(outBytes))
			} else {
				httpEnvs, errs, err := smdClient.GetComponentEndpoints(cli.Token, args...)
				if err != nil {
					return cli.Errorf(cli.CodeNetwork, "failed to get component endpoints from SMD: %w", err)
				}
				// Since smdClient.GetComponentEndpoints does the fetching iteratively, we need to
				// deal with each error that might have occurred.
				var errorsOccurred = false
				for _, e := range errs {
					if e != nil {
						if errors.Is(e, client.UnsuccessfulHTTPError) {
							log.Logger.Error().Err(e).Msg("SMD component endpoint request yielded unsuccessful HTTP response")
						} else {
							log.Logger.Error().Err(e).Msg("failed to get component endpoint")
						}
						errorsOccurred = true
					}
				}

				// Put selected ComponentEndpoints into array and marshal
				type compEp struct {
					ComponentEndpoints []interface{} `json:"ComponentEndpoints" yaml:"ComponentEndpoints"`
				}
				var ceArr []interface{}
				for i, h := range httpEnvs {
					if errs[i] == nil {
						var ce interface{}
						err := json.Unmarshal(h.Body, &ce)
						if err != nil {
							log.Logger.Warn().Err(err).Msg("failed to unmarshal component endpoint")
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
				outBytes, err := client.FormatBody(cesBytes, cli.FormatOutput)
				if err != nil {
					return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
				}
				fmt.Fprint(cli.Ios.Out(), string(outBytes))
			}

			return nil
		},
	}

	// Create flags
	compepGetCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	compepGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return compepGetCmd
}
