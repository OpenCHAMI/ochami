// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package member

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"

	smd_lib "github.com/openchami/ochami/internal/cli/smd"
	"github.com/openchami/ochami/pkg/client/smd"
)

// groupMemberDeleteOptions holds the flag values for the smd group member delete command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type groupMemberDeleteOptions struct {
	NoConfirm bool
}

// runCoreGroupMemberDelete contains the core logic for the smd group member delete command.
// It takes the parsed options and performs the actual work of deleting group members.
func runCoreGroupMemberDelete(cmd *cobra.Command, opts *groupMemberDeleteOptions, args []string, smdClient *smd.SMDClient) error {
	// Ask before attempting deletion unless --no-confirm was passed
	if !opts.NoConfirm {
		log.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
		respDelete, err := cli.Ios.LoopYesNo("Really delete?")
		if err != nil {
			return cli.Errorf(cli.CodeGeneric, "error fetching user input: %w", err)
		} else if !respDelete {
			log.Logger.Info().Msg("User aborted group deletion")
			return nil
		} else {
			log.Logger.Debug().Msg("User answered affirmatively to delete groups members")
		}
	}

	// Handle token for this command
	if err := cli.HandleToken(cmd); err != nil {
		return err
	}

	// Perform deletion from arguments
	results, err := smdClient.DeleteGroupMembers(cmd.Context(), cli.Token, args[0], args[1:]...)
	if err != nil {
		return cli.Errorf(cli.CodeNetwork, "failed to delete members from group %s in SMD: %w", args[0], err)
	}
	// Since smdClient.DeleteGroupMembers does the deletion iteratively, we need to deal with
	// each error that might have occurred.
	var errorsOccurred = false
	for _, e := range results.Errors() {
		if e != nil {
			if errors.Is(e, client.UnsuccessfulHTTPError) {
				log.Logger.Error().Err(e).Msg("SMD group member deletion yielded unsuccessful HTTP response")
			} else {
				log.Logger.Error().Err(e).Msg("failed to delete group member(s)")
			}
			errorsOccurred = true
		}
	}
	// Warn the user if any errors occurred during deletion iterations
	if errorsOccurred {
		return cli.Errorf(cli.CodeHTTP, "SMD group member deletion completed with errors")
	}

	return nil
}

func newCmdGroupMemberDelete() *cobra.Command {
	// groupMemberDeleteCmd represents the "smd group member delete" command
	var groupMemberDeleteCmd = &cobra.Command{
		Use:   "delete <group_label> <component>...",
		Args:  cobra.MinimumNArgs(2),
		Short: "Delete one or more members from a group",
		Long: `Delete one or more members froma group.

See ochami-smd(1) for more details.`,
		Example: `  ochami smd group member delete compute x3000c1s7b56n0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			smdClient, err := smd_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Extract options from flags
			// Since flags are registered with the correct types on this command,
			// these Get* calls cannot fail, so we ignore errors with explicit comments
			opts := &groupMemberDeleteOptions{}
			if cmd.Flag("no-confirm").Changed {
				opts.NoConfirm, _ = cmd.Flags().GetBool("no-confirm") //nolint:errcheck // Flag registered with matching type, error impossible
			}

			return runCoreGroupMemberDelete(cmd, opts, args, smdClient)
		},
	}

	// Create flags
	groupMemberDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return groupMemberDeleteCmd
}
