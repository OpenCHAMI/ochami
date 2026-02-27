// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package node

import (
	"os"

	"github.com/spf13/cobra"

	boot_service_lib "github.com/OpenCHAMI/ochami/internal/cli/boot_service"
	"github.com/OpenCHAMI/ochami/internal/log"

	"github.com/OpenCHAMI/ochami/internal/cli"
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
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			bootServiceClient := boot_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Send off requests
			nodesDeleted, errs, err := bootServiceClient.DeleteNodes(cli.Token, args)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to delete nodes")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Deal with per-request errors
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to delete node")
					errorsOccurred = true
				}
			}
			log.Logger.Debug().Msgf("nodes deleted: %+v", nodesDeleted)
			if errorsOccurred {
				cli.LogHelpError(cmd)
				log.Logger.Warn().Msg("node deletion completed with errors")
				os.Exit(1)
			}
		},
	}

	// Create flags
	bootNodeDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return bootNodeDeleteCmd
}
