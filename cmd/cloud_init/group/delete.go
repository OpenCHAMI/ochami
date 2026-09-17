// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/openchami/cloud-init/pkg/cistore"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"

	cloud_init_lib "github.com/openchami/ochami/internal/cli/cloud_init"
)

func newCmdGroupDelete() *cobra.Command {
	// groupDeleteCmd represents the "cloud-init group delete" command
	var groupDeleteCmd = &cobra.Command{
		Use:   "delete (-d (<data> | @<path>)) | <group>...",
		Short: "Delete one or more cloud-init groups",
		Long: `Delete one or more cloud-init groups. Either one or more group
names must be specified, or raw payload must be specified
with -d. If the argument to -d begins with @, the argument
is interpreted as a file path to read the payload data from.
If the path is -, the data is read from standard input.
-f can be specified to change the format of the input
payload data ('json' by default).

See ochami-cloud-init(1) for more details.`,
		Example: `  # Delete cloud-init groups using CLI arguments
  ochami cloud-init group delete compute my-group

  # Delete cloud-init groups using input payload data
  ochami cloud-init group delete -d '[{"name":"compute"},{"name":"my-group"}]'

  # Delete cloud-init groups using input payload file
  ochami cloud-init group delete -d @payload.json
  ochami cloud-init group delete -d @payload.yaml -f yaml

  # Delete cloud-init groups using data from standard input
  echo '<json_data>' | ochami cloud-init group delete
  echo '<json_data>' | ochami cloud-init group delete -d @-
  echo '<yaml_data>' | ochami cloud-init group delete -f yaml
  echo '<yaml_data>' | ochami cloud-init group delete -d @- -f yaml`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flag("data").Changed {
				if len(args) == 0 {
					return cli.Errorf(cli.CodeUsage, "expected -d or at >= 1 argument (group name(s)); got none")
				}
			} else {
				if len(args) > 0 {
					return cli.Errorf(cli.CodeUsage, "raw data passed, ignoring extra arguments: %v", args)
				}
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// The group data we will send
			ciGroups := []cistore.GroupData{}

			// Read payload from file or stdin.
			var groupsToDel []string
			if cmd.Flag("data").Changed {
				if err := cli.HandlePayload(cmd, &ciGroups); err != nil {
					return err
				}
				for _, group := range ciGroups {
					groupsToDel = append(groupsToDel, group.Name)
				}
			} else {
				groupsToDel = args
			}

			// Ask before attempting deletion unless --no-confirm was passed
			noConfirm, _ := cmd.Flags().GetBool("no-confirm") //nolint:errcheck // flag is registered with the matching type on this command
			if !noConfirm {
				log.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
				respDelete, err := cli.Ios.LoopYesNo("Really delete?")
				if err != nil {
					return cli.Errorf(cli.CodeGeneric, "error fetching user input: %w", err)
				} else if !respDelete {
					log.Logger.Info().Msg("User aborted cloud-init group deletion")
					return nil
				} else {
					log.Logger.Debug().Msg("User answered affirmatively to delete cloud-init groups")
				}
			}

			// Create client to use for requests
			cloudInitClient, err := cloud_init_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// Send data
			results := cloudInitClient.DeleteGroups(cmd.Context(), cli.Token, groupsToDel...)
			// Since the requests are done iteratively, we need to deal with
			// each error that might have occurred.
			var errorsOccurred = false
			for _, e := range results.Errors() {
				if e != nil {
					if errors.Is(e, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(e).Msg("cloud-init group request yielded unsuccessful HTTP response")
					} else {
						log.Logger.Error().Err(e).Msg("failed to delete groups in cloud-init")
					}
					errorsOccurred = true
				}
			}
			if errorsOccurred {
				return cli.Errorf(cli.CodeHTTP, "cloud-init group deletion completed with errors")
			}

			return nil
		},
	}

	// Create flags
	groupDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")
	groupDeleteCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")
	groupDeleteCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")

	groupDeleteCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return groupDeleteCmd
}
