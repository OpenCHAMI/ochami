// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package member

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"

	smd_lib "github.com/openchami/ochami/internal/cli/smd"
)

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
			// Ask before attempting deletion unless --no-confirm was passed
			noConfirm, _ := cmd.Flags().GetBool("no-confirm")
			if !noConfirm {
				log.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
				respDelete, err := cli.Ios.LoopYesNo("Really delete?")
				if err != nil {
					return cli.Errorf(cli.CodeGeneric, "error fetching user input: %w", err)
				} else if !respDelete {
					return cli.Errorf(cli.CodeDeclined, "user aborted group deletion")
				} else {
					log.Logger.Debug().Msg("User answered affirmatively to delete groups members")
				}
			}

			// Create client to use for requests
			smdClient, err := smd_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// Perform deletion from arguments
			results, err := smdClient.DeleteGroupMembers(cmd.Context(), cli.Token, args[0], args[1:]...)
			if err != nil {
				return cli.ClassifyClientError(err, fmt.Sprintf("failed to delete members from group %s in SMD", args[0]), fmt.Sprintf("failed to delete members from group %s in SMD", args[0]))
			}
			if err := cli.AggregateItemErrors(results.Errors(), "SMD group member deletion"); err != nil {
				return err
			}

			return nil
		},
	}

	// Create flags
	groupMemberDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	return groupMemberDeleteCmd
}
