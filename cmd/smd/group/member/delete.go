// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package member

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"

	smd_lib "github.com/OpenCHAMI/ochami/internal/cli/smd"
)

func newCmdGroupMemberDelete() *cobra.Command {
	// groupMemberDeleteCmd represents the "smd group member delete" command
	var groupMemberDeleteCmd = &cobra.Command{
		Use:   "delete <group_label> <component>...",
		Args:  cobra.MinimumNArgs(2),
		Short: "Delete one or more members from a group",
		Long: `Delete one or more members froma group.

See ochami-smd(1) for more details.`,
		Example: `  ochami smd group member delete compute x3000c1s7b56n0`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			smdClient := smd_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Ask before attempting deletion unless --no-confirm was passed
			if !cmd.Flag("no-confirm").Changed {
				log.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
				respDelete, err := cli.Ios.LoopYesNo("Really delete?")
				if err != nil {
					log.Logger.Error().Err(err).Msg("Error fetching user input")
					os.Exit(1)
				} else if !respDelete {
					log.Logger.Info().Msg("User aborted group deletion")
					os.Exit(0)
				} else {
					log.Logger.Debug().Msg("User answered affirmatively to delete groups members")
				}
			}

			// Perform deletion from arguments
			_, errs, err := smdClient.DeleteGroupMembers(cli.Token, args[0], args[1:]...)
			if err != nil {
				log.Logger.Error().Err(err).Msgf("failed to delete members from group %s in SMD", args[0])
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
			// Since smdClient.DeleteGroupMembers does the deletion iteratively, we need to deal with
			// each error that might have occurred.
			var errorsOccurred = false
			for _, e := range errs {
				if err != nil {
					if errors.Is(e, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(e).Msg("SMD group member deletion yielded unsuccessful HTTP response")
					} else {
						log.Logger.Error().Err(e).Msg("failed to delete group member(s)")
					}
					errorsOccurred = true
				}
			}
			// Warn the user if any errors occurred during deletion iterations
			if errorsOccurred {
				cli.LogHelpError(cmd)
				log.Logger.Warn().Msg("SMD group member deletion completed with errors")
				os.Exit(1)
			}
		},
	}

	// Create flags
	groupMemberDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return groupMemberDeleteCmd
}
