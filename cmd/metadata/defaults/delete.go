// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package defaults

import (
	"os"

	"github.com/spf13/cobra"

	metadata_service_lib "github.com/OpenCHAMI/ochami/internal/cli/metadata_service"
	"github.com/OpenCHAMI/ochami/internal/log"

	"github.com/OpenCHAMI/ochami/internal/cli"
)

func newCmdMetadataDefaultsDelete() *cobra.Command {
	// metadataDefaultsDeleteCmd represents the "metadata defaults delete" command
	var metadataDefaultsDeleteCmd = &cobra.Command{
		Use:   "delete <uid>...",
		Args:  cobra.MinimumNArgs(1),
		Short: "Delete one or more cluster defaults",
		Long: `Delete one or more cluster defaults.

See ochami-metadata(1) for more details.`,
		Example: `  # Delete a cluster defaults
  ochami metadata defaults delete clusterdefaults-d614b918

  # Delete multiple cluster defaults
  ochami metadata defaults delete clusterdefaults-d614b918 clusterdefaults-82c40109

  # Don't confirm deletion
  ochami metadata defaults delete --no-confirm clusterdefaults-d614b918`,
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
					log.Logger.Info().Msg("user aborted cluster default deletion")
					os.Exit(0)
				} else {
					log.Logger.Debug().Msg("user answered affirmatively to delete cluster defaults")
				}
			}

			// Send off requests
			defaultsDeleted, errs, err := metadataServiceClient.DeleteDefaults(cli.Token, args)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to delete cluster defaults")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Deal with per-request errors
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to delete cluster defaults")
					errorsOccurred = true
				}
			}

			// Print UIDs of deleted items
			log.Logger.Info().Msgf("Cluster defaults deleted: %+v", defaultsDeleted)

			// Warn if any request errors occurred
			if errorsOccurred {
				cli.LogHelpError(cmd)
				log.Logger.Warn().Msg("cluster defaults deletion completed with errors")
				os.Exit(1)
			}
		},
	}

	// Create flags
	metadataDefaultsDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return metadataDefaultsDeleteCmd
}
