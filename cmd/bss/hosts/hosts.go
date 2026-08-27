// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package hosts

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
)

func NewCmd() *cobra.Command {
	// hostsCmd represents the "bss hosts" command
	var hostsCmd = &cobra.Command{
		Use:   "hosts",
		Args:  cobra.NoArgs,
		Short: "Work with hosts in BSS",
		Long: `Work with hosts in BSS.

See ochami-bss(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cli.PrintUsageHandleError(cmd)
			}
			return nil
		},
	}

	// Add subcommands
	hostsCmd.AddCommand(
		newCmdHostsGet(),
	)

	return hostsCmd
}
