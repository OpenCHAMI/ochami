// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package node

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"

	boot_service_lib "github.com/openchami/ochami/internal/cli/boot_service"
)

// bootNodeDeleteOptions holds the flag values for the boot node delete command.
type bootNodeDeleteOptions struct {
	NoConfirm bool
}

// runCoreBootNodeDelete contains the core logic for the boot node delete command.
// It takes the parsed options and performs the actual work of deleting nodes.
func runCoreBootNodeDelete(cmd *cobra.Command, opts *bootNodeDeleteOptions, args []string, rt *cli.Runtime) error {
	// Ask before attempting deletion unless --no-confirm was passed
	if !opts.NoConfirm {
		rt.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
		respDelete, err := rt.Ios.LoopYesNo("Really delete?")
		if err != nil {
			return cli.Errorf(cli.CodeGeneric, "failed to fetch user input: %w", err)
		} else if !respDelete {
			return cli.Errorf(cli.CodeDeclined, "user aborted node deletion")
		} else {
			rt.Logger.Debug().Msg("user answered affirmatively to delete node(s)")
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
	results := bootServiceClient.DeleteNodes(cmd.Context(), rt.Token, args)

	rt.Logger.Debug().Msgf("nodes deleted: %+v", results.Values())
	if err := cli.AggregateItemErrors(rt.Logger, results.Errors(), "node deletion"); err != nil {
		return err
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
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Extract options from flags
			// Since flags are registered with the correct types on this command,
			// these Get* calls cannot fail, so their errors are ignored
			opts := &bootNodeDeleteOptions{}
			if cmd.Flag("no-confirm").Changed {
				opts.NoConfirm, _ = cmd.Flags().GetBool("no-confirm")
			}

			return runCoreBootNodeDelete(cmd, opts, args, rt)
		},
	}

	// Create flags
	bootNodeDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return bootNodeDeleteCmd
}
