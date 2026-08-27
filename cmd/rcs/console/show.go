// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package console

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/cli/rcs"
)

func newShowCmd() *cobra.Command {
	var follow bool
	var lines int

	var showCmd = &cobra.Command{
		Use:   "show [nodeID]",
		Short: "Shows the console",
		Long: `Shows console output for the specified node.

See ochami-rcs(1) for more details.`,
		Example: `  # Show console output for a node
  ochami rcs console show x0c0s1b0n0`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			follow, err := cmd.Flags().GetBool("follow")
			if err != nil {
				return cli.Errorf(cli.CodeUsage, "unable to get follow flag: %w", err)
			}

			lines, err := cmd.Flags().GetInt("lines")
			if err != nil {
				return cli.Errorf(cli.CodeUsage, "unable to get lines flag: %w", err)
			}

			nodeID := args[0]

			rcsClient, err := rcs.GetClient(cmd)
			if err != nil {
				return err
			}
			err = rcsClient.ShowConsole(cmd.Context(), nodeID, follow, lines, cli.Token, cli.Ios.Out())
			if err != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to show console: %w", err)
			}

			return nil
		},
	}

	showCmd.Flags().BoolVarP(&follow, "follow", "f", false, "follow the console output")
	showCmd.Flags().IntVarP(&lines, "lines", "n", 100, "number of lines to show from history")

	return showCmd
}
