// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package member

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"

	smd_lib "github.com/openchami/ochami/internal/cli/smd"
)

func newCmdGroupMemberAdd() *cobra.Command {
	// groupMemberAddCmd represents the "smd group member add" command
	var groupMemberAddCmd = &cobra.Command{
		Use:   "add <group_label> <component>...",
		Args:  cobra.MinimumNArgs(2),
		Short: "Add one or more components to a group",
		Long: `Add one or more components to a group.

See ochami-smd(1) for more details.`,
		Example: `  ochami smd group member add compute x3000c1s7b56n0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			smdClient, err := smd_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// Send off request
			_, errs, err := smdClient.PostGroupMembers(cli.Token, args[0], args[1:]...)
			if err != nil {
				return cli.ClassifyClientError(err, fmt.Sprintf("failed to add group member(s) to group %s in SMD", args[0]), fmt.Sprintf("failed to add group member(s) to group %s in SMD", args[0]))
			}
			if err := cli.AggregateItemErrors(errs, "SMD group member addition"); err != nil {
				return err
			}

			return nil
		},
	}

	return groupMemberAddCmd
}
