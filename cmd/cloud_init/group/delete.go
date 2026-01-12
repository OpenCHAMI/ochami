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

	"github.com/OpenCHAMI/cloud-init/pkg/cistore"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"

	cloud_init_lib "github.com/OpenCHAMI/ochami/internal/cli/cloud_init"
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
					return fmt.Errorf("expected -d or at >= 1 argument (group name(s)); got none")
				}
			} else {
				if len(args) > 0 {
					return fmt.Errorf("raw data passed, ignoring extra arguments: %v", args)
				}
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			cloudInitClient := cloud_init_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// The group data we will send
			ciGroups := []cistore.GroupData{}

			// Read payload from file or stdin.
			var groupsToDel []string
			if cmd.Flag("data").Changed {
				cli.HandlePayload(cmd, &ciGroups)
				for _, group := range ciGroups {
					groupsToDel = append(groupsToDel, group.Name)
				}
			} else {
				groupsToDel = args
			}

			// Ask before attempting deletion unless --no-confirm was passed
			if !cmd.Flag("no-confirm").Changed {
				log.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
				respDelete, err := cli.Ios.LoopYesNo("Really delete?")
				if err != nil {
					log.Logger.Error().Err(err).Msg("Error fetching user input")
					os.Exit(1)
				} else if !respDelete {
					log.Logger.Info().Msg("User aborted cloud-init group deletion")
					os.Exit(0)
				} else {
					log.Logger.Debug().Msg("User answered affirmatively to delete cloud-init groups")
				}
			}

			// Send data
			_, errs, err := cloudInitClient.DeleteGroups(cli.Token, groupsToDel...)
			if err != nil {
				log.Logger.Error().Err(err).Msgf("failed to delete groups")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
			// Since the requests are done iteratively, we need to deal with
			// each error that might have occurred.
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(err).Msg("cloud-init group request yielded unsuccessful HTTP response")
					} else {
						log.Logger.Error().Err(err).Msg("failed to delete groups in cloud-init")
					}
					errorsOccurred = true
				}
			}
			if errorsOccurred {
				log.Logger.Warn().Msg("cloud-init group deletion completed with errors")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
		},
	}

	// Create flags
	groupDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")
	groupDeleteCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")
	groupDeleteCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")

	groupDeleteCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return groupDeleteCmd
}
