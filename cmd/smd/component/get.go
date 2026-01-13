// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package component

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"

	smd_lib "github.com/OpenCHAMI/ochami/internal/cli/smd"
)

func newCmdComponentGet() *cobra.Command {
	// componentGetCmd represents the "smd component get" command
	var componentGetCmd = &cobra.Command{
		Use:   "get",
		Args:  cobra.NoArgs,
		Short: "Get all components or component identified by an xname or node ID",
		Long: `Get all components or component by an xname or node ID.

See ochami-smd(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			smdClient := smd_lib.GetClient(cmd)

			var httpEnv client.HTTPEnvelope
			var err error
			if cmd.Flag("xname").Changed {
				// This endpoint requires authentication, so a token is needed
				cli.SetToken(cmd)
				cli.CheckToken(cmd)

				httpEnv, err = smdClient.GetComponentsXname(cmd.Flag("xname").Value.String(), cli.Token)
			} else if cmd.Flag("nid").Changed {
				// This endpoint requires authentication, so a token is needed
				cli.SetToken(cmd)
				cli.CheckToken(cmd)

				var nid int32
				nid, err = cmd.Flags().GetInt32("nid")
				if err != nil {
					log.Logger.Error().Err(err).Msg("error getting nid from flag")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}
				httpEnv, err = smdClient.GetComponentsNid(nid, cli.Token)
			} else {
				httpEnv, err = smdClient.GetComponentsAll()
			}
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("SMD component request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to request components from SMD")
				}
				os.Exit(1)
			}

			// Print output
			if outBytes, err := client.FormatBody(httpEnv.Body, cli.FormatOutput); err != nil {
				log.Logger.Error().Err(err).Msg("failed to format output")
				cli.LogHelpError(cmd)
				os.Exit(1)
			} else {
				fmt.Print(string(outBytes))
			}
		},
	}

	// Create flags
	componentGetCmd.Flags().StringP("xname", "x", "", "xname whose Component to fetch")
	componentGetCmd.Flags().Int32P("nid", "n", 0, "node ID whose Component to fetch")
	componentGetCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	componentGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)
	componentGetCmd.MarkFlagsMutuallyExclusive("xname", "nid")

	return componentGetCmd
}
