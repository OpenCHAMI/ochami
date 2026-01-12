// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/smd"

	smd_lib "github.com/OpenCHAMI/ochami/internal/cli/smd"
)

func newCmdGroupAdd() *cobra.Command {
	// groupAddCmd represents the "smd group add" command
	var groupAddCmd = &cobra.Command{
		Use:   "add (-d (<payload_data> | @<payload_file>)) | <group_label>",
		Args:  cobra.MaximumNArgs(1),
		Short: "Add new group",
		Long: `Add new group. A group name is required. Alternatively,
pass -d to pass raw payload data or (if flag argument
starts with @) a file containing the payload data. -f
can be specified to change the format of the input payload
data ('json' by default), but the rules above still
apply for the payload. If "-" is used as the input payload
filename, the data is read from standard input.

This command sends a POST to SMD. An access cli.Token is required.

See ochami-smd(1) for more details.`,
		Example: `  # Add group using CLI flags
  ochami smd group add computes
  ochami smd group add -D "Compute group" computes
  ochami smd group add -D "Compute group" --tag tag1,tag2 --m x3000c1s7b0n1,x3000c1s7b1n1 computes
  ochami smd group add \
    --description "ARM64 group" \
    --tag arm,64-bit \
    --member x3000c1s7b0n1,x3000c1s7b1n1 \
    --exclusive-group amd64 \
    arm64

  # Add groups using input paylad data
  ochami smd group add -d '{[
    {
      "label": "computes",
      "description": "Compute group",
      "tags": ["tag1","tag2"],
      "members": {
        "ids": [
	  "x3000c1s7b0n1",
	  "x3000c1s7b1n1"
	],
      },
    }
  ]}'

  # Add groups using input payload file
  ochami smd group add -d @payload.json
  ochami smd group add -d @payload.yaml -f yaml

  # Add groups using data from standard input
  echo '<json_data>' | ochami smd group add -d @-
  echo '<yaml_data>' | ochami smd group add -d @- -f yaml`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Check that all required args are passed
			if !cmd.Flag("data").Changed {
				if len(args) != 1 {
					return fmt.Errorf("expected -d or 1 argument (group label), got %d", len(args))
				}
			} else {
				if len(args) > 0 {
					log.Logger.Warn().Msgf("raw data passed, ignoring CLI configuration for passed group %v", args)
				}
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			smdClient := smd_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Check if a CA certificate was passed and load it into client if valid
			cli.UseCACert(smdClient.OchamiClient)

			var groups []smd.Group
			var err error
			if cmd.Flag("data").Changed {
				// Use payload file if passed
				cli.HandlePayload(cmd, &groups)
			} else {
				// ...otherwise use CLI options/args
				group := smd.Group{Label: args[0]}
				if cmd.Flag("description").Changed {
					if group.Description, err = cmd.Flags().GetString("description"); err != nil {
						log.Logger.Error().Err(err).Msg("unable to fetch description")
						cli.LogHelpError(cmd)
						os.Exit(1)
					}
				}
				if cmd.Flag("tag").Changed {
					if group.Tags, err = cmd.Flags().GetStringSlice("tag"); err != nil {
						log.Logger.Error().Err(err).Msg("unable to fetch tags")
						cli.LogHelpError(cmd)
						os.Exit(1)
					}
				}
				if cmd.Flag("exclusive-group").Changed {
					if group.ExclusiveGroup, err = cmd.Flags().GetString("exclusive-group"); err != nil {
						log.Logger.Error().Err(err).Msg("unable to fetch exclusive group name")
						cli.LogHelpError(cmd)
						os.Exit(1)
					}
				}
				if cmd.Flag("member").Changed {
					if group.Members.IDs, err = cmd.Flags().GetStringSlice("member"); err != nil {
						log.Logger.Error().Err(err).Msg("unable to fetch members")
						cli.LogHelpError(cmd)
						os.Exit(1)
					}
				}
				groups = append(groups, group)
			}

			// Send off request
			_, errs, err := smdClient.PostGroups(groups, cli.Token)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to add group to SMD")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
			// Since smdClient.PostGroups does the addition iteratively, we need to deal with
			// each error that might have occurred.
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(err).Msg("SMD group request yielded unsuccessful HTTP response")
					} else {
						log.Logger.Error().Err(err).Msg("failed to add group(s) to SMD")
					}
					errorsOccurred = true
				}
			}
			if errorsOccurred {
				cli.LogHelpError(cmd)
				log.Logger.Warn().Msg("SMD group addition completed with errors")
				os.Exit(1)
			}
		},
	}

	// Create flags
	groupAddCmd.Flags().StringP("description", "D", "", "brief description of group")
	groupAddCmd.Flags().StringSlice("tag", []string{}, "one or more tags for group")
	groupAddCmd.Flags().StringP("exclusive-group", "e", "", "name of group that cannot share members with this one")
	groupAddCmd.Flags().StringSliceP("member", "m", []string{}, "one or more component IDs to add to the new group")
	groupAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	groupAddCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	groupAddCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)
	groupAddCmd.MarkFlagsMutuallyExclusive("description", "data")
	groupAddCmd.MarkFlagsMutuallyExclusive("tag", "data")
	groupAddCmd.MarkFlagsMutuallyExclusive("exclusive-group", "data")
	groupAddCmd.MarkFlagsMutuallyExclusive("member", "data")

	return groupAddCmd
}
