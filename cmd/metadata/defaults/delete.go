// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package defaults

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client/metadata_service"

	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
)

// metadataDefaultsDeleteOptions holds the flag values for the metadata defaults delete command.
type metadataDefaultsDeleteOptions struct {
	NoConfirm bool
}

// runCoreMetadataDefaultsDelete contains the core logic for the metadata defaults delete command.
// It takes the parsed options and performs the actual work of deleting cluster defaults.
func runCoreMetadataDefaultsDelete(cmd *cobra.Command, opts *metadataDefaultsDeleteOptions, args []string, metadataServiceClient *metadata_service.MetadataServiceClient, rt *cli.Runtime) error {
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
			return cli.Errorf(cli.CodeDeclined, "user aborted cluster default deletion")
		} else {
			rt.Logger.Debug().Msg("user answered affirmatively to delete cluster defaults")
		}
	}

	// Send off requests
	results := metadataServiceClient.DeleteDefaults(cmd.Context(), rt.Token, args)

	// Print UIDs of deleted items
	rt.Logger.Info().Msgf("Cluster defaults deleted: %+v", results.Values())

	if err := cli.AggregateItemErrors(rt.Logger, results.Errors(), "cluster defaults deletion"); err != nil {
		return err
	}

	return nil
}

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
			opts := &metadataDefaultsDeleteOptions{}
			if cmd.Flag("no-confirm").Changed {
				opts.NoConfirm, _ = cmd.Flags().GetBool("no-confirm")
			}

			return runCoreMetadataDefaultsDelete(cmd, opts, args, metadataServiceClient, rt)
		},
	}

	// Create flags
	metadataDefaultsDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return metadataDefaultsDeleteCmd
}
