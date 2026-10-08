// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package bmc

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	boot_service_lib "github.com/openchami/ochami/internal/cli/boot_service"
	"github.com/openchami/ochami/internal/log"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ask before attempting deletion unless --no-confirm was passed
			noConfirm, _ := cmd.Flags().GetBool("no-confirm")
			if !noConfirm {
				log.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
				respDelete, err := cli.Ios.LoopYesNo("Really delete?")
				if err != nil {
					return cli.Errorf(cli.CodeGeneric, "error fetching user input: %w", err)
				} else if !respDelete {
					return cli.Errorf(cli.CodeDeclined, "user aborted BMC deletion")
				} else {
					log.Logger.Debug().Msg("user answered affirmatively to delete BMC(s)")
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
			results := bootServiceClient.DeleteBMCs(cmd.Context(), cli.Token, args)

			log.Logger.Debug().Msgf("BMCs deleted: %+v", results.Values())
			if err := cli.AggregateItemErrors(results.Errors(), "BMC deletion"); err != nil {
				return err
			}

			return nil
		},
	}

	// Create flags
	bootBmcDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return bootBmcDeleteCmd
}
