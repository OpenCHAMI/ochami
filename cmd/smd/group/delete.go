// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/smd"

	smd_lib "github.com/openchami/ochami/internal/cli/smd"
)

func newCmdGroupDelete() *cobra.Command {
	// groupDeleteCmd represents the "smd group delete" command
	var groupDeleteCmd = &cobra.Command{
		Use:   "delete (-d (<payload_data> | @<payload_file>)) | <group_label>...",
		Short: "Delete one or more groups",
		Long: `Delete one or more groups. These can be specified by one or more group labels.
Alternatively, pass -d to pass raw payload data or (if flag
argument starts with @) a file containing the payload data.
-f can be specified to change the format of the input payload
data ('json' by default), but the rules above still apply for
the payload. If "-" is used as the input payload filename, the
data is read from standard input.

This command sends a DELETE to SMD. An access token is required.

See ochami-smd(1) for more details.`,
		Example: `  # Delete groups using CLI flags
  ochami smd group delete compute

  # Delete groups using input payload data
  ochami smd group delete -d '{[{"label":"compute"}]}'

  # Delete groups using input payload file
  ochami smd group delete -d @payload.json
  ochami smd group delete -d @payload.yaml -f yaml

  # Delete groups using data from standard input
  echo '<json_data>' | ochami smd group delete -d @-
  echo '<yaml_data>' | ochami smd group delete -d @- -f yaml`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// With options, only one of:
			// - A payload file with -d
			// - A set of one or more group labels
			// must be passed.
			if !cmd.Flag("data").Changed {
				if len(args) == 0 {
					return cli.Errorf(cli.CodeUsage, "expected -d or >= 1 argument (group label), got %d", len(args))
				}
			} else {
				if len(args) > 1 {
					log.Logger.Warn().Msgf("raw data passed, ignoring extra arguments: %v", args)
				}
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ask before attempting deletion unless --no-confirm was passed
			noConfirm, err := cmd.Flags().GetBool("no-confirm")
			if err != nil {
				return cli.Errorf(cli.CodeUsage, "failed to get --no-confirm: %w", err)
			}
			if !noConfirm {
				log.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
				respDelete, err := cli.Ios.LoopYesNo("Really delete?")
				if err != nil {
					return cli.Errorf(cli.CodeGeneric, "error fetching user input: %w", err)
				} else if !respDelete {
					log.Logger.Info().Msg("User aborted group deletion")
					return nil
				} else {
					log.Logger.Debug().Msg("User answered affirmatively to delete groups")
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

			// Create list of group labels to delete
			var groups []smd.Group
			var gLabelSlice []string
			if cmd.Flag("data").Changed {
				// Use payload file if passed
				if err := cli.HandlePayload(cmd, &groups); err != nil {
					return err
				}
				for _, group := range groups {
					gLabelSlice = append(gLabelSlice, group.Label)
				}
				if len(gLabelSlice) == 0 {
					return cli.Errorf(cli.CodeUsage, "payload contained no groups to delete")
				}
			} else {
				// ...otherwise, use passed CLI arguments
				gLabelSlice = args
			}

			// Perform deletion
			_, errs, err := smdClient.DeleteGroups(cli.Token, gLabelSlice...)
			if err != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to delete groups in SMD: %w", err)
			}
			// Since smdClient.DeleteGroups does the deletion iteratively, we need to deal with
			// each error that might have occurred.
			var errorsOccurred = false
			for _, e := range errs {
				if e != nil {
					if errors.Is(e, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(e).Msg("SMD group deletion yielded unsuccessful HTTP response")
					} else {
						log.Logger.Error().Err(e).Msg("failed to delete group")
					}
					errorsOccurred = true
				}
			}
			// Warn the user if any errors occurred during deletion iterations
			if errorsOccurred {
				return cli.Errorf(cli.CodeHTTP, "SMD group deletion completed with errors")
			}

			return nil
		},
	}

	// Create flags
	groupDeleteCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	groupDeleteCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")
	groupDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	groupDeleteCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return groupDeleteCmd
}
