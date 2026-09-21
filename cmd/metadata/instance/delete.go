// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package instance

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client/metadata_service"

	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
)

// metadataInstanceDeleteOptions holds the flag values for the metadata instance delete command.
type metadataInstanceDeleteOptions struct {
	NoConfirm bool
}

// runCoreMetadataInstanceDelete contains the core logic for the metadata instance delete command.
// It takes the parsed options and performs the actual work of deleting instance infos.
func runCoreMetadataInstanceDelete(cmd *cobra.Command, opts *metadataInstanceDeleteOptions, args []string, metadataServiceClient *metadata_service.MetadataServiceClient, rt *cli.Runtime) error {
	// Handle token for this command
	if err := rt.HandleToken(cmd); err != nil {
		return err
	}

	// Ask before attempting deletion unless --no-confirm was passed
	if !opts.NoConfirm {
		rt.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
		respDelete, err := rt.Ios.LoopYesNo("Really delete?")
		if err != nil {
			return cli.Errorf(cli.CodeGeneric, "error fetching user input: %w", err)
		} else if !respDelete {
			return cli.Errorf(cli.CodeDeclined, "user aborted instance info deletion")
		} else {
			rt.Logger.Debug().Msg("user answered affirmatively to delete instance infos")
		}
	}

	// Send off requests
	results := metadataServiceClient.DeleteInstanceInfos(cmd.Context(), rt.Token, args)

	// Print UIDs of deleted items
	rt.Logger.Info().Msgf("Instance infos deleted: %+v", results.Values())

	if err := cli.AggregateItemErrors(rt.Logger, results.Errors(), "Instance info deletion"); err != nil {
		return err
	}

	return nil
}

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
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Create client to use for requests
			metadataServiceClient, err := metadata_service_lib.GetClient(cmd, rt)
			if err != nil {
				return err
			}

			// Extract options from flags
			// Since flags are registered with the correct types on this command,
			// these Get* calls cannot fail, so their errors are ignored
			opts := &metadataInstanceDeleteOptions{}
			if cmd.Flag("no-confirm").Changed {
				opts.NoConfirm, _ = cmd.Flags().GetBool("no-confirm")
			}

			return runCoreMetadataInstanceDelete(cmd, opts, args, metadataServiceClient, rt)
		},
	}

	// Create flags
	metadataInstanceDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return metadataInstanceDeleteCmd
}
