// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package bmc

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"

	boot_service_lib "github.com/openchami/ochami/internal/cli/boot_service"
)

// bootBmcDeleteOptions holds the flag values for the boot bmc delete command.
type bootBmcDeleteOptions struct {
	NoConfirm bool
}

// runCoreBootBmcDelete contains the core logic for the boot bmc delete command.
// It takes the parsed options and performs the actual work of deleting BMCs.
func runCoreBootBmcDelete(cmd *cobra.Command, opts *bootBmcDeleteOptions, args []string, rt *cli.Runtime) error {
	// Ask before attempting deletion unless --no-confirm was passed
	if !opts.NoConfirm {
		rt.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
		respDelete, err := rt.Ios.LoopYesNo("Really delete?")
		if err != nil {
			return cli.Errorf(cli.CodeGeneric, "error fetching user input: %w", err)
		} else if !respDelete {
			return cli.Errorf(cli.CodeDeclined, "user aborted BMC deletion")
		} else {
			rt.Logger.Debug().Msg("user answered affirmatively to delete BMC(s)")
		}
	}

	// Create client to use for requests
	bootServiceClient, err := boot_service_lib.GetClient(cmd, rt)
	if err != nil {
		return err
	}

	// Handle token for this command
	if err := rt.HandleToken(cmd); err != nil {
		return err
	}

	// Send off requests
	results := bootServiceClient.DeleteBMCs(cmd.Context(), rt.Token, args)

	rt.Logger.Debug().Msgf("BMCs deleted: %+v", results.Values())
	if err := cli.AggregateItemErrors(rt.Logger, results.Errors(), "BMC deletion"); err != nil {
		return err
	}

	return nil
}

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
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Extract options from flags
			// Since flags are registered with the correct types on this command,
			// these Get* calls cannot fail, so their errors are ignored
			opts := &bootBmcDeleteOptions{}
			if cmd.Flag("no-confirm").Changed {
				opts.NoConfirm, _ = cmd.Flags().GetBool("no-confirm")
			}

			return runCoreBootBmcDelete(cmd, opts, args, rt)
		},
	}

	// Create flags
	bootBmcDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return bootBmcDeleteCmd
}
