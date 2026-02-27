// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package boot

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/config"

	// Subcommands
	bmc_cmd "github.com/OpenCHAMI/ochami/cmd/boot/bmc"
	config_cmd "github.com/OpenCHAMI/ochami/cmd/boot/config"
	node_cmd "github.com/OpenCHAMI/ochami/cmd/boot/node"
)

func NewCmd() *cobra.Command {
	// bootCmd represents the boot command
	var bootCmd = &cobra.Command{
		Use:   "boot",
		Args:  cobra.NoArgs,
		Short: "Communicate with the boot service",
		Long: `Communicate with the boot service.

See ochami-boot(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
				os.Exit(0)
			}
		},
	}

	// Create flags
	bootCmd.PersistentFlags().String("api-version", "", "version of service API to use in request")
	bootCmd.PersistentFlags().Duration("timeout", config.DefaultConfig.Timeout, "timeout duration when making requests")
	bootCmd.PersistentFlags().String("uri", "", "absolute base URI or relative base path of boot service")

	// Add subcommands
	bootCmd.AddCommand(
		bmc_cmd.NewCmd(),
		config_cmd.NewCmd(),
		node_cmd.NewCmd(),
	)

	return bootCmd
}
