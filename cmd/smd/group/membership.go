// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"

	smd_lib "github.com/OpenCHAMI/ochami/internal/cli/smd"
)

func newCmdGroupMembership() *cobra.Command {
	// groupMembershipCmd represents the "smd group membership" command
	var groupMembershipCmd = &cobra.Command{
		Use:   "membership --id <xname>",
		Args:  cobra.NoArgs,
		Short: "Get all group memberships of one or more nodes",
		Long: `Get all group memberships of one or more nodes.

See ochami-smd(1) for more details.`,
		Example: `  # Get group membership for all nodes,
  ochami smd group membership

  # Get group membership for a subset of nodes (two)
  ochami smd group membership x1000c0s0b0n0 x1000c0s1b0n0

  # Get group membership for nodes whose IDs are between 1000
  # and 2000 and are of x86 architecture
  ochami smd group membership --nid-start 1000 --nid-end 2000 --arch X86`,
		Run: func(cmd *cobra.Command, args []string) {

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
				values, err := cmd.Flags().GetStringArray(flag)
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to parse flags")
					cli.LogHelpError(cmd)
					os.Exit(1)
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
						log.Logger.Error().Err(err).Msg("failed to parse flags")
						cli.LogHelpError(cmd)
						os.Exit(1)
					}
					params.Add(strings.ReplaceAll(flag, "-", "_"), value)
				}
			}

			// Create client to use for requests
			smdClient := smd_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			httpEnv, err := smdClient.GetGroupMembership(params.Encode(), cli.Token)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("SMD membership request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to request membership from SMD")
				}
				cli.LogHelpError(cmd)
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
	groupMembershipCmd.Flags().StringArray("id", nil, "filter the results based on xname ID(s)")
	groupMembershipCmd.Flags().StringArray("type", nil, "filter the results based on HMS type")
	groupMembershipCmd.Flags().StringArray("state", nil, "filter the results based on HMS state")
	groupMembershipCmd.Flags().StringArray("flag", nil, "filter the results based on HMS flag value")
	groupMembershipCmd.Flags().StringArray("role", nil, "filter the results based on HMS role")
	groupMembershipCmd.Flags().StringArray("subrole", nil, "filter the results based on HMS subrole")
	groupMembershipCmd.Flags().StringArray("softwarestatus", nil, "filter the results based on software status")
	groupMembershipCmd.Flags().StringArray("subtype", nil, "filter the results based on HMS subtype")
	groupMembershipCmd.Flags().StringArray("arch", nil, "filter the results based on architecture")
	groupMembershipCmd.Flags().StringArray("class", nil, "filter the results based on HMS hardware class")
	groupMembershipCmd.Flags().StringArray("nid", nil, "filter the results based on NID")
	groupMembershipCmd.Flags().String("enabled", "", "filter the results based on enabled status")
	groupMembershipCmd.Flags().String("nid-start", "", "filter the results based on NIDs equal to or greater than the provided integer")
	groupMembershipCmd.Flags().String("nid-end", "", "filter the results based on NIDs less than or equal to the provided integer")
	groupMembershipCmd.Flags().String("partition", "", "restrict search to the given partition")
	groupMembershipCmd.Flags().String("group", "", "restrict search to the given group label")

	groupMembershipCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	groupMembershipCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return groupMembershipCmd
}
