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

func newCmdGroupMemberAdd() *cobra.Command {
	// groupMemberAddCmd represents the "smd group member add" command
	var groupMemberAddCmd = &cobra.Command{
		Use:   "add <group_label> <component>...",
		Args:  cobra.MinimumNArgs(2),
		Short: "Add one or more components to a group",
		Long: `Add one or more components to a group.

See ochami-smd(1) for more details.`,
		Example: `  ochami smd group member add compute x3000c1s7b56n0`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			smdClient := smd_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Send off request
			_, errs, err := smdClient.PostGroupMembers(cli.Token, args[0], args[1:]...)
			if err != nil {
				log.Logger.Error().Err(err).Msgf("failed to add group member(s) to group %s in SMD", args[0])
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
			// Since smdClient.PostGroupMembers does the addition iteratively, we need to deal with
			// each error that might have occurred.
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(err).Msgf("SMD group member request for group %s yielded unsuccessful HTTP response", args[0])
					} else {
						log.Logger.Error().Err(err).Msgf("failed to add group member(s) to group %s in SMD", args[0])
					}
					errorsOccurred = true
				}
			}
			if errorsOccurred {
				cli.LogHelpError(cmd)
				log.Logger.Warn().Msg("SMD group addition completed with errors")
				os.Exit(1)
			}
		},
	}

	return groupMemberAddCmd
}
