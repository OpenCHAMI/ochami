// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package member

import (
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

			// Send request
			httpEnv, err := smdClient.GetGroupMembers(cmd.Context(), args[0], rt.Token)
			if err != nil {
				return cli.ClassifyClientError(err, "SMD group member request yielded unsuccessful HTTP response", "failed to request group members from SMD")
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

	cli.AddFormatOutputFlag(groupMemberGetCmd)
	groupMemberGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return groupMemberGetCmd
}
