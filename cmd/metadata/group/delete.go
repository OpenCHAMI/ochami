// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	metadata_service_lib "github.com/OpenCHAMI/ochami/internal/cli/metadata_service"
	"github.com/OpenCHAMI/ochami/internal/log"
)

func newCmdMetadataGroupDelete() *cobra.Command {
	// metadataGroupDeleteCmd represents the "metadata group delete" command
	var metadataGroupDeleteCmd = &cobra.Command{
		Use:   "delete <uid>...",
		Args:  cobra.MinimumNArgs(1),
		Short: "Delete one or more groups",
		Long: `Delete one or more groups.

See ochami-metadata(1) for more details.`,
		Example: `  # Delete a group
  ochami metadata group delete group-d614b918

  # Delete multiple groups
  ochami metadata group delete group-d614b918 group-82c40109

  # Don't confirm deletion
  ochami metadata group delete --no-confirm group-d614b918`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			metadataServiceClient := metadata_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Ask before attempting deletion unless --no-confirm was passed
			if !cmd.Flag("no-confirm").Changed {
				log.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
				respDelete, err := cli.Ios.LoopYesNo("Really delete?")
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to fetch user input")
					os.Exit(1)
				} else if !respDelete {
					log.Logger.Info().Msg("user aborted group deletion")
					os.Exit(0)
				} else {
					log.Logger.Debug().Msg("user answered affirmatively to delete groups")
				}
			}

			// Send off requests
			groupsDeleted, errs, err := metadataServiceClient.DeleteGroups(cli.Token, args)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to delete groups")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Deal with per-request errors
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to delete group")
					errorsOccurred = true
				}
			}

			// Print UIDs of deleted items
			log.Logger.Info().Msgf("Groups deleted: %+v", groupsDeleted)

			// Warn if any request errors occurred
			if errorsOccurred {
				cli.LogHelpError(cmd)
				log.Logger.Warn().Msg("Group deletion completed with errors")
				os.Exit(1)
			}
		},
	}

	// Create flags
	metadataGroupDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return metadataGroupDeleteCmd
}
