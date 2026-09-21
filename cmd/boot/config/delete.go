// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client/boot_service"

	boot_service_lib "github.com/openchami/ochami/internal/cli/boot_service"
)

// bootConfigDeleteOptions holds the flag values for the boot config delete command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type bootConfigDeleteOptions struct {
	NoConfirm bool
}

// runCoreBootConfigDelete contains the core logic for the boot config delete command.
// It takes the parsed options and performs the actual work of deleting boot configurations.
func runCoreBootConfigDelete(cmd *cobra.Command, opts *bootConfigDeleteOptions, args []string, bootServiceClient *boot_service.BootServiceClient, rt *cli.Runtime) error {
	// Ask before attempting deletion unless --no-confirm was passed
	if !opts.NoConfirm {
		rt.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
		respDelete, err := rt.Ios.LoopYesNo("Really delete?")
		if err != nil {
			return cli.Errorf(cli.CodeGeneric, "failed to fetch user input: %w", err)
		} else if !respDelete {
			rt.Logger.Info().Msg("user aborted boot config deletion")
			return nil
		} else {
			rt.Logger.Debug().Msg("user answered affirmatively to delete boot config(s)")
		}
	}

	// Handle token for this command
	if err := rt.HandleToken(cmd); err != nil {
		return err
	}

	// Send off requests
	results := bootServiceClient.DeleteBootConfigs(cmd.Context(), rt.Token, args)

	// Deal with per-request errors
	var errorsOccurred = false
	for _, e := range results.Errors() {
		if e != nil {
			rt.Logger.Error().Err(e).Msg("failed to delete boot config")
			errorsOccurred = true
		}
	}
	rt.Logger.Debug().Msgf("boot configs deleted: %+v", results.Values())
	if errorsOccurred {
		return cli.Errorf(cli.CodeHTTP, "boot config deletion completed with errors")
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
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Create client to use for requests
			bootServiceClient, err := boot_service_lib.GetClientWithRuntime(cmd, rt)
			if err != nil {
				return err
			}

			// Extract options from flags
			// Since flags are registered with the correct types on this command,
			// these Get* calls cannot fail, so we ignore errors with explicit comments
			opts := &bootConfigDeleteOptions{}
			if cmd.Flag("no-confirm").Changed {
				opts.NoConfirm, _ = cmd.Flags().GetBool("no-confirm") //nolint:errcheck // Flag registered with matching type, error impossible
			}

			return runCoreBootConfigDelete(cmd, opts, args, bootServiceClient, rt)
		},
	}

	// Create flags
	bootConfigDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return bootConfigDeleteCmd
}
