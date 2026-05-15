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
		Run: func(cmd *cobra.Command, args []string) {
			cli.HandleToken(cmd)

			follow, err := cmd.Flags().GetBool("follow")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to get follow flag")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			lines, err := cmd.Flags().GetInt("lines")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to get lines flag")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			nodeID := args[0]

			rcsClient := rcs.GetClient(cmd)
			err = rcsClient.ShowConsole(cmd.Context(), nodeID, follow, lines, cli.Token, os.Stdout)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to show console")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
		},
	}

	showCmd.Flags().BoolVarP(&follow, "follow", "f", false, "follow the console output")
	showCmd.Flags().IntVarP(&lines, "lines", "n", 100, "number of lines to show from history")

	return showCmd
}
