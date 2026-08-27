// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package console

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/cli/rcs"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			nodeID := args[0]
			rcsClient, err := rcs.GetClient(cmd)
			if err != nil {
				return err
			}
			err = rcsClient.ConnectConsole(cmd.Context(), nodeID, cli.Token, cli.Ios.In(), cli.Ios.Out())
			if err != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to connect to console: %w", err)
			}

			return nil
		},
	}
}
