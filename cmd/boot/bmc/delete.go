// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package bmc

import (
	"os"

	"github.com/spf13/cobra"

	boot_service_lib "github.com/OpenCHAMI/ochami/internal/cli/boot_service"
	"github.com/OpenCHAMI/ochami/internal/log"

	"github.com/OpenCHAMI/ochami/internal/cli"
)

func newCmdBootBmcDelete() *cobra.Command {
	// bootBmcDeleteCmd represents the "boot bmc delete" command
	var bootBmcDeleteCmd = &cobra.Command{
		Use:   "delete <uid>...",
		Args:  cobra.MinimumNArgs(1),
		Short: "Delete one or more BMCs",
		Long: `Delete one or more BMCs.

See ochami-boot(1) for more details.`,
		Example: `  # Delete a BMC
  ochami boot bmc delete bmc-773d99bf

  # Delete multiple BMCs
  ochami boot bmc delete bmc-773d99bf bmc-773d99c0

  # Don't confirm deletion
  ochami boot bmc delete --no-confirm bmc-773d99bf`,
		Run: func(cmd *cobra.Command, args []string) {
			// Ask before attempting deletion unless --no-confirm was passed
			noConfirm, err := cmd.Flags().GetBool("no-confirm")
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to get --no-confirm")
				os.Exit(1)
			}
			if !noConfirm {
				log.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
				respDelete, err := cli.Ios.LoopYesNo("Really delete?")
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to fetch user input")
					os.Exit(1)
				} else if !respDelete {
					log.Logger.Info().Msg("user aborted BMC deletion")
					os.Exit(0)
				} else {
					log.Logger.Debug().Msg("user answered affirmatively to delete BMC(s)")
				}
			}

			// Create client to use for requests
			bootServiceClient := boot_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Send off requests
			bmcsDeleted, errs, err := bootServiceClient.DeleteBMCs(cli.Token, args)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to delete BMCs")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Deal with per-request errors
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to delete BMC")
					errorsOccurred = true
				}
			}
			log.Logger.Debug().Msgf("BMCs deleted: %+v", bmcsDeleted)
			if errorsOccurred {
				cli.LogHelpError(cmd)
				log.Logger.Warn().Msg("BMC deletion completed with errors")
				os.Exit(1)
			}
		},
	}

	// Create flags
	bootBmcDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return bootBmcDeleteCmd
}
