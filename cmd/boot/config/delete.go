// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"

	boot_service_lib "github.com/openchami/ochami/internal/cli/boot_service"
)

// bootConfigDeleteOptions holds the flag values for the boot config delete command.
type bootConfigDeleteOptions struct {
	NoConfirm bool
}

// runCoreBootConfigDelete contains the core logic for the boot config delete command.
// It takes the parsed options and performs the actual work of deleting boot configs.
func runCoreBootConfigDelete(cmd *cobra.Command, opts *bootConfigDeleteOptions, args []string) error {
	// Ask before attempting deletion unless --no-confirm was passed
	if !opts.NoConfirm {
		log.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
		respDelete, err := cli.Ios.LoopYesNo("Really delete?")
		if err != nil {
			return cli.Errorf(cli.CodeGeneric, "failed to fetch user input: %w", err)
		} else if !respDelete {
			return cli.Errorf(cli.CodeDeclined, "user aborted boot config deletion")
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
	results := bootServiceClient.DeleteBootConfigs(cmd.Context(), cli.Token, args)

	log.Logger.Debug().Msgf("boot configs deleted: %+v", results.Values())
	if err := cli.AggregateItemErrors(results.Errors(), "boot config deletion"); err != nil {
		return err
	}

	return nil
}

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
			// Extract options from flags
			// Since flags are registered with the correct types on this command,
			// these Get* calls cannot fail, so their errors are ignored
			opts := &bootConfigDeleteOptions{}
			if cmd.Flag("no-confirm").Changed {
				opts.NoConfirm, _ = cmd.Flags().GetBool("no-confirm")
			}

			return runCoreBootConfigDelete(cmd, opts, args)
		},
	}

	// Create flags
	bootConfigDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return bootConfigDeleteCmd
}
