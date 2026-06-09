// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package peer

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	metadata_service_lib "github.com/OpenCHAMI/ochami/internal/cli/metadata_service"
	"github.com/OpenCHAMI/ochami/internal/log"
)

func newCmdMetadataPeerDelete() *cobra.Command {
	// metadataPeerDeleteCmd represents the "metadata peer delete" command
	var metadataPeerDeleteCmd = &cobra.Command{
		Use:   "delete <uid>...",
		Args:  cobra.MinimumNArgs(1),
		Short: "Delete one or more WireGuard peers",
		Long: `Delete one or more WireGuard peers.

See ochami-metadata(1) for more details.`,
		Example: `  # Delete a WireGuard peer
  ochami metadata peer delete wireguardpeer-d614b918

  # Delete multiple WireGuard peers
  ochami metadata peer delete wireguardpeer-d614b918 wireguardpeer-82c40109

  # Don't confirm deletion
  ochami metadata peer delete --no-confirm wireguardpeer-d614b918`,
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
					log.Logger.Info().Msg("user aborted WireGuard peer deletion")
					os.Exit(0)
				} else {
					log.Logger.Debug().Msg("user answered affirmatively to delete WireGuard peers")
				}
			}

			// Send off requests
			peersDeleted, errs, err := metadataServiceClient.DeleteWireGuardPeers(cli.Token, args)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to delete WireGuard peers")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Deal with per-request errors
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to delete WireGuard peer")
					errorsOccurred = true
				}
			}
			log.Logger.Debug().Msgf("WireGuard peers deleted: %+v", peersDeleted)
			if errorsOccurred {
				cli.LogHelpError(cmd)
				log.Logger.Warn().Msg("WireGuard peer deletion completed with errors")
				os.Exit(1)
			}
		},
	}

	// Create flags
	metadataPeerDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return metadataPeerDeleteCmd
}
