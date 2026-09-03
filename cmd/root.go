// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/internal/version"

	// Subcommands
	boot_cmd "github.com/openchami/ochami/cmd/boot"
	bss_cmd "github.com/openchami/ochami/cmd/bss"
	cloud_init_cmd "github.com/openchami/ochami/cmd/cloud_init"
	config_cmd "github.com/openchami/ochami/cmd/config"
	discover_cmd "github.com/openchami/ochami/cmd/discover"
	metadata_cmd "github.com/openchami/ochami/cmd/metadata"
	pcs_cmd "github.com/openchami/ochami/cmd/pcs"
	rcs_cmd "github.com/openchami/ochami/cmd/rcs"
	smd_cmd "github.com/openchami/ochami/cmd/smd"
	version_cmd "github.com/openchami/ochami/cmd/version"
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
		// Errors and usage are handled centrally in Execute so that failures
		// produce a single, consistent message plus a help hint and a
		// differentiated exit code. SilenceErrors/SilenceUsage prevent Cobra
		// from also printing them.
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Ask the user in any child commands to create the config file
			// if missing. If this is undesired, define PersistentPreRunE in
			// the child command with this line overridden with:
			//
			//   cli.InitConfigAndLogging(cmd, false)
			//
			if err := cli.InitConfigAndLogging(cmd, true); err != nil {
				return err
			}

			// Apply the default formats (if the flags aren't changed and the config option is present)
			// Note that this doesn't cover the case where the variable is checked without the corresponding
			// flag being defined.
			inputFormatFlag := cmd.Flag("format-input")
			if cli.ActiveConfig().DefaultInputFormat != "" && inputFormatFlag != nil && !inputFormatFlag.Changed {
				cli.FormatInput = cli.ActiveConfig().DefaultInputFormat
			}

			outputFormatFlag := cmd.Flag("format-output")
			if cli.ActiveConfig().DefaultOutputFormat != "" && outputFormatFlag != nil && !outputFormatFlag.Changed {
				cli.FormatOutput = cli.ActiveConfig().DefaultOutputFormat
			}

			return nil
		},
		// With no subcommand, print usage and exit successfully.
		RunE: cli.PrintUsage,
	}

	// Create root command flags
	rootCmd.PersistentFlags().StringVarP(&cli.ConfigFile, "config", "c", "", "path to configuration file to use")
	rootCmd.PersistentFlags().StringP("log-format", "L", "", "log format (json,rfc3339,basic)")
	rootCmd.PersistentFlags().StringP("log-level", "l", "", "set verbosity of logs (info,warning,debug)")
	rootCmd.PersistentFlags().String("log-color", "", "set coloring of logs (auto,on,off)")
	rootCmd.PersistentFlags().StringP("cluster", "C", "", "name of cluster whose config to use for this command")
	rootCmd.PersistentFlags().StringP("cluster-uri", "u", "", "base URI for OpenCHAMI services, excluding service base path (overrides cluster.uri in config file)")
	rootCmd.PersistentFlags().StringVar(&cli.CACertPath, "cacert", "", "path to root CA certificate in PEM format")
	rootCmd.PersistentFlags().StringVarP(&cli.Token, "token", "t", "", "access cli.Token to present for authentication")
	rootCmd.PersistentFlags().Bool("no-token", false, "do not check for or use an access cli.Token")
	rootCmd.PersistentFlags().Bool("show-token", false, "show full access token in debug logs instead of a truncated value")
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
		metadata_cmd.NewCmd(),
		pcs_cmd.NewCmd(),
		rcs_cmd.NewCmd(),
		smd_cmd.NewCmd(),
		version_cmd.NewCmd(),
	)

	// Ensure flag-parse and argument-validation errors across the whole
	// command tree resolve to the CodeUsage exit code.
	cli.WrapUsageErrors(rootCmd)

	return rootCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd := NewRootCmd()
	err := rootCmd.Execute()
	if err != nil {
		log.Logger.Error().Err(err).Msg("failed to execute command")
		if cmd, _, ferr := rootCmd.Find(os.Args[1:]); ferr != nil {
			// Error looking up invoked command, default to printing
			// help suggestion for root command, printing debug
			// message only for debugging (most users don't need to
			// know an error occurred).
			log.Logger.Debug().Err(ferr).Msg("failed to lookup invoked command")
			cli.LogHelpHint(rootCmd)
		} else {
			// Print help suggestion for invoked command
			cli.LogHelpHint(cmd)
		}
		os.Exit(cli.ExitCode(err))
	}
}
