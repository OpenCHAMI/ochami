// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package instance

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
	"github.com/openchami/ochami/internal/log"
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
					log.Logger.Info().Msg("user aborted instance info deletion")
					return nil
				} else {
					log.Logger.Debug().Msg("user answered affirmatively to delete instance infos")
				}
			}

			// Send off requests
			instancesDeleted, errs, err := metadataServiceClient.DeleteInstanceInfos(cli.Token, args)
			if err != nil {
				return cli.ClassifyClientError(err, "failed to delete instance infos", "failed to delete instance infos")

			}

			// Deal with per-request errors
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to delete instance info")
					errorsOccurred = true
				}
			}

			// Print UIDs of deleted items
			log.Logger.Info().Msgf("Instance infos deleted: %+v", instancesDeleted)

			// Warn if any request errors occurred
			if errorsOccurred {
				return cli.Errorf(cli.CodeHTTP, "Instance info deletion completed with errors")
			}

			return nil
		},
	}

	// Create flags
	metadataInstanceDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return metadataInstanceDeleteCmd
}
