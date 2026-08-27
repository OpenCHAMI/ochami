// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package node

import (
	"github.com/spf13/cobra"

	boot_service_lib "github.com/openchami/ochami/internal/cli/boot_service"
	"github.com/openchami/ochami/internal/log"

	"github.com/openchami/ochami/internal/cli"
)

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
			// Ask before attempting deletion unless --no-confirm was passed
			noConfirm, err := cmd.Flags().GetBool("no-confirm")
			if err != nil {
				return cli.Errorf(cli.CodeUsage, "failed to get --no-confirm: %w", err)
			}
			if !noConfirm {
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

			// Create client to use for requests
			bootServiceClient, err := boot_service_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// Send off requests
			nodesDeleted, errs, err := bootServiceClient.DeleteNodes(cli.Token, args)
			if err != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to delete nodes: %w", err)
			}

			// Deal with per-request errors
			var errorsOccurred = false
			for _, e := range errs {
				if e != nil {
					log.Logger.Error().Err(e).Msg("failed to delete node")
					errorsOccurred = true
				}
			}
			log.Logger.Debug().Msgf("nodes deleted: %+v", nodesDeleted)
			if errorsOccurred {
				return cli.Errorf(cli.CodeHTTP, "node deletion completed with errors")
			}

			return nil
		},
	}

	// Create flags
	bootNodeDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return bootNodeDeleteCmd
}
