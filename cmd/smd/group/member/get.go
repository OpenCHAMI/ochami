// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package member

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	smd_lib "github.com/openchami/ochami/internal/cli/smd"
)

func newCmdGroupMemberGet() *cobra.Command {
	// groupMemberGetCmd represents the "smd group member get" command
	var groupMemberGetCmd = &cobra.Command{
		Use:     "get <group_label>",
		Aliases: []string{"list"},
		Args:    cobra.ExactArgs(1),
		Short:   "Get members of a group",
		Long: `Get members of a group.

See ochami-smd(1) for more details.`,
		Example: `  ochami smd group member get compute`,
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

			// Send request
			httpEnv, err := smdClient.GetGroupMembers(args[0], cli.Token)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					return cli.Errorf(cli.CodeHTTP, "SMD group member request yielded unsuccessful HTTP response: %w", err)
				}
				return cli.Errorf(cli.CodeNetwork, "failed to request group members from SMD: %w", err)
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
	groupMemberGetCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	groupMemberGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return groupMemberGetCmd
}
