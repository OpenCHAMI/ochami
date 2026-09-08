// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
	"github.com/openchami/ochami/internal/log"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			metadataServiceClient, err := metadata_service_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// Ask before attempting deletion unless --no-confirm was passed
			noConfirm, _ := cmd.Flags().GetBool("no-confirm") //nolint:errcheck // flag is registered with the matching type on this command
			if !noConfirm {
				log.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
				respDelete, err := cli.Ios.LoopYesNo("Really delete?")
				if err != nil {
					return cli.Errorf(cli.CodeGeneric, "error fetching user input: %w", err)
				} else if !respDelete {
					log.Logger.Info().Msg("user aborted group deletion")
					return nil
				} else {
					log.Logger.Debug().Msg("user answered affirmatively to delete groups")
				}
			}

			// Send off requests
			groupsDeleted, errs, err := metadataServiceClient.DeleteGroups(cli.Token, args)
			if err != nil {
				return cli.ClassifyClientError(err, "failed to delete groups", "failed to delete groups")

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
				return cli.Errorf(cli.CodeHTTP, "Group deletion completed with errors")
			}

			return nil
		},
	}

	// Create flags
	metadataGroupDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return metadataGroupDeleteCmd
}
