// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"os"

	"github.com/spf13/cobra"

	boot_service_lib "github.com/OpenCHAMI/ochami/internal/cli/boot_service"
	"github.com/OpenCHAMI/ochami/internal/log"

	"github.com/OpenCHAMI/ochami/internal/cli"
)

func newCmdBootConfigDelete() *cobra.Command {
	// bootConfigDeleteCmd represents the "boot config delete" command
	var bootConfigDeleteCmd = &cobra.Command{
		Use:   "delete <uid>...",
		Args:  cobra.MinimumNArgs(1),
		Short: "Delete one or more boot configs",
		Long: `Delete one or more boot configs.

See ochami-boot(1) for more details.`,
		Example: `  # Delete a boot configuration
  ochami boot config delete boo-ebf2a27a

  # Delete multiple boot configurations
  ochami boot config delete boo-ebf2a27a boo-ebf2a27b

  # Don't confirm deletion
  ochami boot config delete --no-confirm boo-ebf2a27a`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			bootServiceClient := boot_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Send off requests
			bcfgsDeleted, errs, err := bootServiceClient.DeleteBootConfigs(cli.Token, args)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to delete boot configs")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Deal with per-request errors
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to delete boot config")
					errorsOccurred = true
				}
			}
			log.Logger.Debug().Msgf("boot configs deleted: %+v", bcfgsDeleted)
			if errorsOccurred {
				cli.LogHelpError(cmd)
				log.Logger.Warn().Msg("boot config deletion completed with errors")
				os.Exit(1)
			}
		},
	}

	// Create flags
	bootConfigDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return bootConfigDeleteCmd
}
