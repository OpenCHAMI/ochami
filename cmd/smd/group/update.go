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

func newCmdGroupUpdate() *cobra.Command {
	// groupUpdateCmd represents the "smd group update" command
	var groupUpdateCmd = &cobra.Command{
		Use:   "update (-d (<payload_data> | @<payload_file>)) | ([--description <description>] [--tag <tag>]... <group_label>)",
		Args:  cobra.MaximumNArgs(1),
		Short: "Update the description and/or tags of a group",
		Long: `Update the description and/or tags of a group. At least one of --description
or --tag must be specified. Alternatively, pass -d to pass
raw payload data or (if flag argument starts with @) a file
containing the payload data. -f can be specified to change
the format of the input payload data ('json' by default), but
the rules above still apply for the payload. If "-" is used as
the input payload filename, the data is read from standard input.

This command sends a PATCH to SMD. An access cli.Token is required.

See ochami-smd(1) for more details.`,
		Example: `  # Update a group using CLI flags
  ochami smd group update --description "New description for compute" compute
  ochami smd group update --tag existing_tag --tag new_tag compute
  ochami smd group update --tag existing_tag,new_tag compute
  ochami smd group update --tag existing_tag,new_tag -D "New description for compute" compute

  # Update groups using input payload data
  ochami smd group update -d '{[
    {
      "label": "compute",
      "description": "New compute group description"
    }
  ]}'

  # Update groups using input payload file
  ochami smd group update -d @payload.json
  ochami smd group update -d @payload.yaml -f yaml

  # Update groups using data from standard input
  echo '<json_data>' | ochami smd group update -d @-
  echo '<yaml_data>' | ochami smd group update -d @- -f yaml`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// cmd.LocalFlags().NFlag() doesn't seem to work, so we check every flag
			if !cmd.Flag("data").Changed {
				if len(args) == 0 {
					return fmt.Errorf("expected -d or >= 1 argument (group label), got %d", len(args))
				} else {
					if !cmd.Flag("description").Changed && !cmd.Flag("tag").Changed {
						return fmt.Errorf("group label passed, but no --description/--tag (at least one is required)")
					}
				}
			} else {
				if len(args) > 0 {
					log.Logger.Warn().Msgf("raw data passed, ignoring extra arguments: %v", args)
				}
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			smdClient := smd_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// The group list we will send
			var groups []smd.Group

			// Read payload from file first, allowing overwrites from flags
			var err error
			if cmd.Flag("data").Changed {
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
				groups = append(groups, group)
			}

			// Send 'em off
			_, errs, err := smdClient.PatchGroups(groups, cli.Token)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to patch group in SMD")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
			// Since smdClient.PatchGroups does the edition iteratively, we need to deal with
			// each error that might have occurred.
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(err).Msg("SMD group request yielded unsuccessful HTTP response")
					} else {
						log.Logger.Error().Err(err).Msg("failed to update group(s) to SMD")
					}
					errorsOccurred = true
				}
			}
			if errorsOccurred {
				log.Logger.Warn().Msg("SMD group update completed with errors")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
		},
	}

	// Create flags
	groupUpdateCmd.Flags().StringP("description", "D", "", "short description to update group with")
	groupUpdateCmd.Flags().StringSlice("tag", []string{}, "one or more tags to set for group")
	groupUpdateCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	groupUpdateCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	groupUpdateCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)
	groupUpdateCmd.MarkFlagsOneRequired("description", "tag", "data")

	return groupUpdateCmd
}
