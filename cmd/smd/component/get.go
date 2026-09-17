// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package component

import (
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

			var httpEnv client.HTTPEnvelope
			if cmd.Flag("xname").Changed {
				// This endpoint requires authentication, so a token is needed
				if err := rt.HandleToken(cmd); err != nil {
					return err
				}

				httpEnv, err = smdClient.GetComponentsXname(cmd.Context(), cmd.Flag("xname").Value.String(), rt.Token)
			} else if cmd.Flag("nid").Changed {
				// This endpoint requires authentication, so a token is needed
				if err := rt.HandleToken(cmd); err != nil {
					return err
				}

				var nid int32
				nid, err = cmd.Flags().GetInt32("nid")
				if err != nil {
					return cli.Errorf(cli.CodeUsage, "error getting nid from flag: %w", err)
				}
				httpEnv, err = smdClient.GetComponentsNid(cmd.Context(), nid, rt.Token)
			} else {
				httpEnv, err = smdClient.GetComponentsAll(cmd.Context())
			}
			if err != nil {
				return cli.ClassifyClientError(err, "SMD component request yielded unsuccessful HTTP response", "failed to request components from SMD")
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
	componentGetCmd.Flags().StringP("xname", "x", "", "xname whose Component to fetch")
	componentGetCmd.Flags().Int32P("nid", "n", 0, "node ID whose Component to fetch")

	cli.AddFormatOutputFlag(componentGetCmd)
	componentGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)
	componentGetCmd.MarkFlagsMutuallyExclusive("xname", "nid")

	return componentGetCmd
}
