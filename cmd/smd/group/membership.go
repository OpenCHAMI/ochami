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

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"

	smd_lib "github.com/OpenCHAMI/ochami/internal/cli/smd"
)

func newCmdGroupMembership() *cobra.Command {
	// groupGetCmd represents the "smd group get" command

	var groupMembershipCmd = &cobra.Command{
		Use:   "membership <node>",
		Args:  cobra.NoArgs,
		Short: "Get all group memberships of a node",
		Long: `Get all group memberships of a node.

See ochami-smd(1) for more details.`,
		Example: `  ochami smd group get x1000c0s0b0n0`,
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
					params.Add(flag, v)
				}
			}

			for _, flag := range []string{
				"enabled",
				"nid_start",
				"nid_end",
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
					params.Add(flag, value)
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

	groupMembershipCmd.Flags().StringArray("id", nil, "Filter the results based on xname ID(s). Can be specified multiple times for selecting entries with multiple specific xnames.")
	groupMembershipCmd.Flags().StringArray("type", nil, "Filter the results based on HMS type like Node, NodeEnclosure, NodeBMC etc. Can be specified multiple times for selecting entries of multiple types.")
	groupMembershipCmd.Flags().StringArray("state", nil, "Filter the results based on HMS state like Ready, On etc. Can be specified multiple times for selecting entries in different states.")
	groupMembershipCmd.Flags().StringArray("flag", nil, "Filter the results based on HMS flag value like OK, Alert etc. Can be specified multiple times for selecting entries with different flags.")
	groupMembershipCmd.Flags().StringArray("role", nil, "Filter the results based on HMS role. Can be specified multiple times for selecting entries with different roles. Valid values are:, Compute, Service, System, Application, Storage, Management. Additional valid values may be added via configuration file. See the results of 'GET /service/values/role' for the complete list.")
	groupMembershipCmd.Flags().StringArray("subrole", nil, "Filter the results based on HMS subrole. Can be specified multiple times for selecting entries with different subroles. Valid values are:, Master, Worker, Storage. Additional valid values may be added via configuration file. See the results of 'GET /service/values/subrole' for the complete list.")
	groupMembershipCmd.Flags().StringArray("softwarestatus", nil, "Filter the results based on software status. Software status is a free form string. Matching is case-insensitive. Can be specified multiple times for selecting entries with different software statuses.")
	groupMembershipCmd.Flags().StringArray("subtype", nil, "Filter the results based on HMS subtype. Can be specified multiple times for selecting entries with different subtypes.")
	groupMembershipCmd.Flags().StringArray("arch", nil, "Filter the results based on architecture. Can be specified multiple times for selecting components with different architectures.")
	groupMembershipCmd.Flags().StringArray("class", nil, "Filter the results based on HMS hardware class. Can be specified multiple times for selecting entries with different classes.")
	groupMembershipCmd.Flags().StringArray("nid", nil, "Filter the results based on NID. Can be specified multiple times for selecting entries with multiple specific NIDs.")
	groupMembershipCmd.Flags().String("enabled", "", "Filter the results based on enabled status (true or false).")
	groupMembershipCmd.Flags().String("nid_start", "", "Filter the results based on NIDs equal to or greater than the provided integer.")
	groupMembershipCmd.Flags().String("nid_end", "", "Filter the results based on NIDs less than or equal to the provided integer.")
	groupMembershipCmd.Flags().String("partition", "", "Restrict search to the given partition (p#.#). One partition can be combined with at most one group argument which will be treated as a logical AND. NULL will return components in NO partition.")
	groupMembershipCmd.Flags().String("group", "", "Restrict search to the given group label. One group can be combined with at most one partition argument which will be treated as a logical AND. NULL will return components in NO groups.")

	groupMembershipCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	groupMembershipCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return groupMembershipCmd
}
