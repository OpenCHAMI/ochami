// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package defaults

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
	"github.com/openchami/ochami/internal/log"
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
					log.Logger.Info().Msg("user aborted cluster default deletion")
					return nil
				} else {
					log.Logger.Debug().Msg("user answered affirmatively to delete cluster defaults")
				}
			}

			// Send off requests
			defaultsDeleted, errs, err := metadataServiceClient.DeleteDefaults(cli.Token, args)
			if err != nil {
				return cli.ClassifyClientError(err, "failed to delete cluster defaults", "failed to delete cluster defaults")

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
				return cli.Errorf(cli.CodeHTTP, "cluster defaults deletion completed with errors")
			}

			return nil
		},
	}

	// Create flags
	metadataDefaultsDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return metadataDefaultsDeleteCmd
}
