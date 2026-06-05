// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package instance

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	metadata_service_lib "github.com/OpenCHAMI/ochami/internal/cli/metadata_service"
	"github.com/OpenCHAMI/ochami/internal/log"
)

func newCmdMetadataInstanceDelete() *cobra.Command {
	// metadataInstanceDeleteCmd represents the "metadata instance delete" command
	var metadataInstanceDeleteCmd = &cobra.Command{
		Use:   "delete <uid>...",
		Args:  cobra.MinimumNArgs(1),
		Short: "Delete one or more instance infos",
		Long: `Delete one or more instance infos.

See ochami-metadata(1) for more details.`,
		Example: `  # Delete an instance info
  ochami metadata instance delete instanceinfo-d614b918

  # Delete multiple instance infos
  ochami metadata instance delete instanceinfo-d614b918 instanceinfo-82c40109

  # Don't confirm deletion
  ochami metadata instance delete --no-confirm instanceinfo-d614b918`,
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
					log.Logger.Info().Msg("user aborted instance info deletion")
					os.Exit(0)
				} else {
					log.Logger.Debug().Msg("user answered affirmatively to delete instance infos")
				}
			}

			// Send off requests
			instancesDeleted, errs, err := metadataServiceClient.DeleteInstanceInfos(cli.Token, args)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to delete instance infos")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Deal with per-request errors
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to delete instance info")
					errorsOccurred = true
				}
			}
			log.Logger.Debug().Msgf("Instance infos deleted: %+v", instancesDeleted)
			if errorsOccurred {
				cli.LogHelpError(cmd)
				log.Logger.Warn().Msg("Instance info deletion completed with errors")
				os.Exit(1)
			}
		},
	}

	// Create flags
	metadataInstanceDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return metadataInstanceDeleteCmd
}
