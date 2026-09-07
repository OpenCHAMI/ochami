// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package component

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	smd_lib "github.com/openchami/ochami/internal/cli/smd"
)

func newCmdComponentGet() *cobra.Command {
	// componentGetCmd represents the "smd component get" command
	var componentGetCmd = &cobra.Command{
		Use:     "get",
		Aliases: []string{"list"},
		Args:    cobra.NoArgs,
		Short:   "Get all components or component identified by an xname or node ID",
		Long: `Get all components or component by an xname or node ID.

See ochami-smd(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			smdClient, err := smd_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			var httpEnv client.HTTPEnvelope
			if cmd.Flag("xname").Changed {
				// This endpoint requires authentication, so a token is needed
				if err := cli.SetToken(cmd); err != nil {
					return err
				}
				if err := cli.CheckToken(cmd); err != nil {
					return err
				}

				httpEnv, err = smdClient.GetComponentsXname(cmd.Flag("xname").Value.String(), cli.Token)
			} else if cmd.Flag("nid").Changed {
				// This endpoint requires authentication, so a token is needed
				if err := cli.SetToken(cmd); err != nil {
					return err
				}
				if err := cli.CheckToken(cmd); err != nil {
					return err
				}

				var nid int32
				nid, err = cmd.Flags().GetInt32("nid")
				if err != nil {
					return cli.Errorf(cli.CodeUsage, "error getting nid from flag: %w", err)
				}
				httpEnv, err = smdClient.GetComponentsNid(nid, cli.Token)
			} else {
				httpEnv, err = smdClient.GetComponentsAll()
			}
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					return cli.Errorf(cli.CodeHTTP, "SMD component request yielded unsuccessful HTTP response: %w", err)
				}
				return cli.Errorf(cli.CodeNetwork, "failed to request components from SMD: %w", err)
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
	componentGetCmd.Flags().StringP("xname", "x", "", "xname whose Component to fetch")
	componentGetCmd.Flags().Int32P("nid", "n", 0, "node ID whose Component to fetch")
	componentGetCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	componentGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)
	componentGetCmd.MarkFlagsMutuallyExclusive("xname", "nid")

	return componentGetCmd
}
