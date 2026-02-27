// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/internal/version"

	// Subcommands
	boot_cmd "github.com/OpenCHAMI/ochami/cmd/boot"
	bss_cmd "github.com/OpenCHAMI/ochami/cmd/bss"
	cloud_init_cmd "github.com/OpenCHAMI/ochami/cmd/cloud_init"
	config_cmd "github.com/OpenCHAMI/ochami/cmd/config"
	discover_cmd "github.com/OpenCHAMI/ochami/cmd/discover"
	pcs_cmd "github.com/OpenCHAMI/ochami/cmd/pcs"
	smd_cmd "github.com/OpenCHAMI/ochami/cmd/smd"
	version_cmd "github.com/OpenCHAMI/ochami/cmd/version"
)

var (
	logLevel  string
	logFormat string
)

func NewRootCmd() *cobra.Command {
	// rootCmd represents the base command when called without any subcommands
	var rootCmd = &cobra.Command{
		Use:   version.ProgName,
		Args:  cobra.NoArgs,
		Short: "Command line interface for interacting with OpenCHAMI services",
		Long: `Command line interface for interacting with OpenCHAMI services.

See ochami(1) for more details on available commands.
See ochami-config(1) for more details on how to configure ochami using the CLI.
See ochami-config(5) for more details on configuring the ochami config file(s).`,
		Version: version.Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Ask the user in any child commands to create the config file
			// if missing. If this is undesired, define PersistentPreRunE in
			// the child command with this line overridden with:
			//
			//   cli.InitConfigAndLogging(cmd, false)
			//
			cli.InitConfigAndLogging(cmd, true)

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
				os.Exit(0)
			}
		},
	}

	// Create root command flags
	rootCmd.PersistentFlags().StringVarP(&cli.ConfigFile, "config", "c", "", "path to configuration file to use")
	rootCmd.PersistentFlags().StringP("log-format", "L", "", "log format (json,rfc3339,basic)")
	rootCmd.PersistentFlags().StringP("log-level", "l", "", "set verbosity of logs (info,warning,debug)")
	rootCmd.PersistentFlags().StringP("cluster", "C", "", "name of cluster whose config to use for this command")
	rootCmd.PersistentFlags().StringP("cluster-uri", "u", "", "base URI for OpenCHAMI services, excluding service base path (overrides cluster.uri in config file)")
	rootCmd.PersistentFlags().StringVar(&cli.CACertPath, "cacert", "", "path to root CA certificate in PEM format")
	rootCmd.PersistentFlags().StringVarP(&cli.Token, "token", "t", "", "access cli.Token to present for authentication")
	rootCmd.PersistentFlags().Bool("no-token", false, "do not check for or use an access cli.Token")
	rootCmd.PersistentFlags().BoolVarP(&cli.Insecure, "insecure", "k", false, "do not verify TLS certificates")
	rootCmd.PersistentFlags().Bool("ignore-config", false, "do not use any config file")
	rootCmd.PersistentFlags().BoolVarP(&log.EarlyLogger.EarlyVerbose, "verbose", "v", false, "be verbose before logging is initialized")

	// Either use cluster from config file or specify details on CLI
	rootCmd.MarkFlagsMutuallyExclusive("cluster", "cluster-uri")

	// Do not allow simultaneously passing a token and ignoring it
	rootCmd.MarkFlagsMutuallyExclusive("token", "no-token")

	// Add subcommands
	rootCmd.AddCommand(
		boot_cmd.NewCmd(),
		bss_cmd.NewCmd(),
		cloud_init_cmd.NewCmd(),
		config_cmd.NewCmd(),
		discover_cmd.NewCmd(),
		pcs_cmd.NewCmd(),
		version_cmd.NewCmd(),
		smd_cmd.NewCmd(),
	)

	return rootCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd := NewRootCmd()
	err := rootCmd.Execute()
	if err != nil {
		log.Logger.Error().Err(err).Msg("failed to execute command")
		if cmd, _, err := rootCmd.Find(os.Args[1:]); err != nil {
			// Error looking up invoked command, default to printing
			// help suggestion for root command, printing debug
			// message only for debugging (most users don't need to
			// know an error occurred).
			log.Logger.Debug().Err(err).Msg("failed to lookup invoked command")
			cli.LogHelpError(rootCmd)
		} else {
			// Print help suggestion for invoked command
			cli.LogHelpError(cmd)
		}
		os.Exit(1)
	}
}
