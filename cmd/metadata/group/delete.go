// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client/metadata_service"

	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
)

// metadataGroupDeleteOptions holds the flag values for the metadata group delete command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type metadataGroupDeleteOptions struct {
	NoConfirm bool
}

// runCoreMetadataGroupDelete contains the core logic for the metadata group delete command.
// It takes the parsed options and performs the actual work of deleting groups.
func runCoreMetadataGroupDelete(cmd *cobra.Command, opts *metadataGroupDeleteOptions, args []string, metadataServiceClient *metadata_service.MetadataServiceClient) error {
	// Handle token for this command
	if err := cli.HandleToken(cmd); err != nil {
		return err
	}

	// Ask before attempting deletion unless --no-confirm was passed
	if !opts.NoConfirm {
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
	results := metadataServiceClient.DeleteGroups(cmd.Context(), cli.Token, args)

	// Deal with per-request errors
	var errorsOccurred = false
	for _, err := range results.Errors() {
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to delete group")
			errorsOccurred = true
		}
	}

	// Print UIDs of deleted items
	log.Logger.Info().Msgf("Groups deleted: %+v", results.Values())

	// Warn if any request errors occurred
	if errorsOccurred {
		return cli.Errorf(cli.CodeHTTP, "Group deletion completed with errors")
	}

	return nil
}

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

			// Extract options from flags
			// Since flags are registered with the correct types on this command,
			// these Get* calls cannot fail, so we ignore errors with explicit comments
			opts := &metadataGroupDeleteOptions{}
			if cmd.Flag("no-confirm").Changed {
				opts.NoConfirm, _ = cmd.Flags().GetBool("no-confirm") //nolint:errcheck // Flag registered with matching type, error impossible
			}

			return runCoreMetadataGroupDelete(cmd, opts, args, metadataServiceClient)
		},
	}

	// Create flags
	metadataGroupDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return metadataGroupDeleteCmd
}
