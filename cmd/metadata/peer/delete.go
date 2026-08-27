// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package peer

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"
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
					log.Logger.Info().Msg("user aborted WireGuard peer deletion")
					return nil
				} else {
					log.Logger.Debug().Msg("user answered affirmatively to delete WireGuard peers")
				}
			}

			// Send off requests
			peersDeleted, errs, err := metadataServiceClient.DeleteWireGuardPeers(cli.Token, args)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					return cli.Errorf(cli.CodeHTTP, "failed to delete WireGuard peers: %w", err)
				}
				return cli.Errorf(cli.CodeNetwork, "failed to delete WireGuard peers: %w", err)
			}

			// Deal with per-request errors
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(err).Msg("failed to delete WireGuard peer")
					} else {
						log.Logger.Error().Err(err).Msg("failed to delete WireGuard peer")
					}
					errorsOccurred = true
				}
			}

			// Print UIDs of deleted items
			log.Logger.Info().Msgf("WireGuard peers deleted: %+v", peersDeleted)

			// Warn if any request errors occurred
			if errorsOccurred {
				return cli.Errorf(cli.CodeHTTP, "WireGuard peer deletion completed with errors")
			}

			return nil
		},
	}

	// Create flags
	metadataPeerDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return metadataPeerDeleteCmd
}
