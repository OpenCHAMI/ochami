// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package member

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"

	smd_lib "github.com/OpenCHAMI/ochami/internal/cli/smd"
)

func newCmdGroupMemberSet() *cobra.Command {
	// groupMemberSetCmd represents the "smd group member set" command
	var groupMemberSetCmd = &cobra.Command{
		Use:   "set <group_label> <component>...",
		Args:  cobra.MinimumNArgs(2),
		Short: "Set group membership list to a list of components",
		Long: `Set group membership list to a list of components. The components specified
in the list are set as the only members of the group. If a component
specified is already in the group, it remains in the group. If a
component specified is not already in te group, it is added to the
group. If a component is in the group but not specified, it is
removed from the group.

See ochami-smd(1) for more details.`,
		Example: `  ochami smd group member set compute x1000c1s7b1n0 x1000c1s7b2n0`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			smdClient := smd_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Send off request
			_, err := smdClient.PutGroupMembers(cli.Token, args[0], args[1:]...)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msgf("SMD group member request for group %s yielded unsuccessful HTTP response", args[0])
				} else {
					log.Logger.Error().Err(err).Msgf("failed to set group membership for group %s in SMD", args[0])
				}
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
		},
	}

	return groupMemberSetCmd
}
