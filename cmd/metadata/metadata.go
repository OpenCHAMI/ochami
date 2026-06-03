// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/config"

	// Subcommands
	service_cmd "github.com/OpenCHAMI/ochami/cmd/metadata/service"
)

func NewCmd() *cobra.Command {
	// metadataCmd represents the metadata command
	var metadataCmd = &cobra.Command{
		Use: "metadata",
		Aliases: []string{
			"md",
		},
		Args:  cobra.NoArgs,
		Short: "Communicate with the metadatat service",
		Long: `Communicate with the metadata service.

See ochami-metadata(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
				os.Exit(0)
			}
		},
	}

	// Create flags
	metadataCmd.PersistentFlags().String("api-version", "", "version of service API to use in request")
	metadataCmd.PersistentFlags().Duration("timeout", config.DefaultConfig.Timeout, "timeout duration when making requests")
	metadataCmd.PersistentFlags().String("uri", "", "absolute base URI or relative base path of boot service")

	// Add subcommands
	metadataCmd.AddCommand(
		service_cmd.NewCmd(),
	)

	return metadataCmd
}
