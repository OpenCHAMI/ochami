// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package console

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/cli/rcs"
	"github.com/OpenCHAMI/ochami/internal/log"
)

func newConnectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "connect [nodeID]",
		Short: "Connects to a console",
		Long: `Connects to an interactive console session on the specified node.

See ochami-rcs(1) for more details.`,
		Example: `  # Connect to a node console
  ochami rcs console connect x0c0s1b0n0`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cli.HandleToken(cmd)

			nodeID := args[0]
			rcsClient := rcs.GetClient(cmd)
			err := rcsClient.ConnectConsole(cmd.Context(), nodeID, cli.Token, os.Stdin, os.Stdout)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to connect to console")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
		},
	}
}
