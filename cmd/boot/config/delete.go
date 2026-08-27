// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"github.com/spf13/cobra"

	boot_service_lib "github.com/openchami/ochami/internal/cli/boot_service"
	"github.com/openchami/ochami/internal/log"

	"github.com/openchami/ochami/internal/cli"
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
					log.Logger.Info().Msg("user aborted boot config deletion")
					return nil
				} else {
					log.Logger.Debug().Msg("user answered affirmatively to delete boot config(s)")
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
			bcfgsDeleted, errs, err := bootServiceClient.DeleteBootConfigs(cli.Token, args)
			if err != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to delete boot configs: %w", err)
			}

			// Deal with per-request errors
			var errorsOccurred = false
			for _, e := range errs {
				if e != nil {
					log.Logger.Error().Err(e).Msg("failed to delete boot config")
					errorsOccurred = true
				}
			}
			log.Logger.Debug().Msgf("boot configs deleted: %+v", bcfgsDeleted)
			if errorsOccurred {
				return cli.Errorf(cli.CodeHTTP, "boot config deletion completed with errors")
			}

			return nil
		},
	}

	// Create flags
	bootConfigDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return bootConfigDeleteCmd
}
