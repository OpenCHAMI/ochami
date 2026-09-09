// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	smd_lib "github.com/openchami/ochami/internal/cli/smd"
)

func newCmdGroupMembership() *cobra.Command {
	// groupMembershipCmd represents the "smd group membership" command
	var groupMembershipCmd = &cobra.Command{
		Use:   "membership",
		Args:  cobra.NoArgs,
		Short: "Get all group memberships of one or more nodes",
		Long: `Get all group memberships of one or more nodes.

See ochami-smd(1) for more details.`,
		Example: `  # Get group membership for all nodes,
  ochami smd group membership

  # Get group membership for a subset of nodes (two)
  ochami smd group membership --id x1000c0s0b0n0 --id x1000c0s1b0n0
  ochami smd group membership --id x1000c0s0b0n0,x1000c0s1b0n0

  # Get group membership for nodes whose IDs are between 1000
  # and 2000 and are of x86 architecture
  ochami smd group membership --nid-start 1000 --nid-end 2000 --arch X86`,
		RunE: func(cmd *cobra.Command, args []string) error {

			params := url.Values{}
			for _, flag := range []string{
				"id",
				"type",
				"state",
				"flag",
				"role",
				"subrole",
				"softwarestatus",
				"subtype",
				"arch",
				"class",
				"nid",
			} {
				values, err := cmd.Flags().GetStringSlice(flag)
				if err != nil {
					return cli.Errorf(cli.CodeUsage, "failed to parse flags: %w", err)
				}
				for _, v := range values {
					params.Add(strings.ReplaceAll(flag, "-", "_"), v)
				}
			}

			for _, flag := range []string{
				"enabled",
				"nid-start",
				"nid-end",
				"partition",
				"group",
			} {
				if cmd.Flags().Changed(flag) {
					value, err := cmd.Flags().GetString(flag)
					if err != nil {
						return cli.Errorf(cli.CodeUsage, "failed to parse flags: %w", err)
					}
					params.Add(strings.ReplaceAll(flag, "-", "_"), value)
				}
			}

			// Create client to use for requests
			smdClient, err := smd_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			httpEnv, err := smdClient.GetGroupMembership(cmd.Context(), params.Encode(), cli.Token)
			if err != nil {
				return cli.ClassifyClientError(err, "SMD membership request yielded unsuccessful HTTP response", "failed to request membership from SMD")

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
	groupMembershipCmd.Flags().StringSlice("id", nil, "filter the results based on xname ID(s)")
	groupMembershipCmd.Flags().StringSlice("type", nil, "filter the results based on HMS type")
	groupMembershipCmd.Flags().StringSlice("state", nil, "filter the results based on HMS state")
	groupMembershipCmd.Flags().StringSlice("flag", nil, "filter the results based on HMS flag value")
	groupMembershipCmd.Flags().StringSlice("role", nil, "filter the results based on HMS role")
	groupMembershipCmd.Flags().StringSlice("subrole", nil, "filter the results based on HMS subrole")
	groupMembershipCmd.Flags().StringSlice("softwarestatus", nil, "filter the results based on software status")
	groupMembershipCmd.Flags().StringSlice("subtype", nil, "filter the results based on HMS subtype")
	groupMembershipCmd.Flags().StringSlice("arch", nil, "filter the results based on architecture")
	groupMembershipCmd.Flags().StringSlice("class", nil, "filter the results based on HMS hardware class")
	groupMembershipCmd.Flags().StringSlice("nid", nil, "filter the results based on NID")
	groupMembershipCmd.Flags().String("enabled", "", "filter the results based on enabled status")
	groupMembershipCmd.Flags().String("nid-start", "", "filter the results based on NIDs equal to or greater than the provided integer")
	groupMembershipCmd.Flags().String("nid-end", "", "filter the results based on NIDs less than or equal to the provided integer")
	groupMembershipCmd.Flags().String("partition", "", "restrict search to the given partition")
	groupMembershipCmd.Flags().String("group", "", "restrict search to the given group label")

	groupMembershipCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	groupMembershipCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return groupMembershipCmd
}
