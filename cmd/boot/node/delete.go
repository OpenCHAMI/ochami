// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package node

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client/boot_service"

	boot_service_lib "github.com/openchami/ochami/internal/cli/boot_service"
)

// bootNodeDeleteOptions holds the flag values for the boot node delete command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type bootNodeDeleteOptions struct {
	NoConfirm bool
}

// runCoreBootNodeDelete contains the core logic for the boot node delete command.
// It takes the parsed options and performs the actual work of deleting nodes.
func runCoreBootNodeDelete(cmd *cobra.Command, opts *bootNodeDeleteOptions, args []string, bootServiceClient *boot_service.BootServiceClient) error {
	// Ask before attempting deletion unless --no-confirm was passed
	if !opts.NoConfirm {
		log.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
		respDelete, err := cli.Ios.LoopYesNo("Really delete?")
		if err != nil {
			return cli.Errorf(cli.CodeGeneric, "failed to fetch user input: %w", err)
		} else if !respDelete {
			log.Logger.Info().Msg("user aborted node deletion")
			return nil
		} else {
			log.Logger.Debug().Msg("user answered affirmatively to delete node(s)")
		}
	}

	// Handle token for this command
	if err := cli.HandleToken(cmd); err != nil {
		return err
	}

	// Send off requests
	results := bootServiceClient.DeleteNodes(cmd.Context(), cli.Token, args)

	// Deal with per-request errors
	var errorsOccurred = false
	for _, e := range results.Errors() {
		if e != nil {
			log.Logger.Error().Err(e).Msg("failed to delete node")
			errorsOccurred = true
		}
	}
	log.Logger.Debug().Msgf("nodes deleted: %+v", results.Values())
	if errorsOccurred {
		return cli.Errorf(cli.CodeHTTP, "node deletion completed with errors")
	}

	return nil
}

func newCmdBootNodeDelete() *cobra.Command {
	// bootNodeDeleteCmd represents the "boot node delete" command
	var bootNodeDeleteCmd = &cobra.Command{
		Use:   "delete <uid>...",
		Args:  cobra.MinimumNArgs(1),
		Short: "Delete one or more nodes",
		Long: `Delete one or more nodes.

See ochami-boot(1) for more details.`,
		Example: `  # Delete a node
  ochami boot node delete nod-bc76f7f2

  # Delete multiple nodes
  ochami boot node delete nod-bc76f7f2 nod-bc76f7f3

  # Don't confirm deletion
  ochami boot node delete --no-confirm nod-bc76f7f2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			bootServiceClient, err := boot_service_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Extract options from flags
			// Since flags are registered with the correct types on this command,
			// these Get* calls cannot fail, so we ignore errors with explicit comments
			opts := &bootNodeDeleteOptions{}
			if cmd.Flag("no-confirm").Changed {
				opts.NoConfirm, _ = cmd.Flags().GetBool("no-confirm") //nolint:errcheck // Flag registered with matching type, error impossible
			}

			return runCoreBootNodeDelete(cmd, opts, args, bootServiceClient)
		},
	}

	// Create flags
	bootNodeDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return bootNodeDeleteCmd
}
