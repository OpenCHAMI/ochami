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
	// Create runtime instance for this command tree
	rt := cli.NewRuntime()

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
			// Check if a runtime was already injected into the context (e.g., by tests)
			// If so, use that runtime but still sync flag values to it
			useGlobalSync := true
			if existingRt, ok := cli.FromContext(cmd.Context()); ok {
				rt = existingRt
				useGlobalSync = false
			}

			// Sync runtime from global state (flags are bound to global variables)
			// This ensures that if tests have set global state via SetIOStream() or
			// flag parsing, the runtime picks it up.
			// For test-injected runtimes, we only sync the flag values (ConfigFile, CACertPath, Token, Insecure)
			// to the runtime, but skip the I/O streams and config initialization to preserve test isolation.
			if useGlobalSync {
				rt.Ios = cli.NewIOStreams(cli.Ios.In(), cli.Ios.Out(), cli.Ios.Err())
				rt.ConfigFile = cli.ConfigFile
				rt.CACertPath = cli.CACertPath
				rt.Token = cli.Token
				rt.Insecure = cli.Insecure
			} else {
				// For test-injected runtimes, only sync flag values that were explicitly set
				// This allows tests to use --config, --token, etc. flags while maintaining isolation
				if cli.ConfigFile != "" {
					rt.ConfigFile = cli.ConfigFile
				}
				if cli.CACertPath != "" {
					rt.CACertPath = cli.CACertPath
				}
				if cli.Token != "" {
					rt.Token = cli.Token
				}
				if cli.Insecure {
					rt.Insecure = cli.Insecure
				}
			}

			// Initialize config and logging using global functions (for backward compatibility)
			// This will populate the global activeConfig, which runtime can access
			// For test-injected runtimes, we only load config if a config file was explicitly provided
			// (via --config flag), to support tests that need config loading while maintaining isolation
			if useGlobalSync {
				if err := cli.InitConfigAndLogging(cmd, true); err != nil {
					return err
				}

				// Sync runtime config from global state after initialization
				rt.Config = cli.ActiveConfig()
				rt.Koanf = cli.ActiveKoanf()
			} else if rt.ConfigFile != "" {
				// For test-injected runtimes with an explicit config file, load the config
				// This allows tests to use --config flag while maintaining isolation for other state
				if err := cli.InitConfigAndLogging(cmd, true); err != nil {
					return err
				}

				// Sync runtime config from global state after initialization
				rt.Config = cli.ActiveConfig()
				rt.Koanf = cli.ActiveKoanf()
			}

			// Apply the default formats (if the flags aren't changed and the config option is present)
			// Note that this doesn't cover the case where the variable is checked without the corresponding
			// flag being defined.
			// For test-injected runtimes, we still apply default formats from the runtime's config if available
			inputFormatFlag := cmd.Flag("format-input")
			if !useGlobalSync && rt.Config.DefaultInputFormat != "" && inputFormatFlag != nil && !inputFormatFlag.Changed {
				rt.FormatInput = rt.Config.DefaultInputFormat
				// Don't update global for test-injected runtime
			} else if useGlobalSync && rt.Config.DefaultInputFormat != "" && inputFormatFlag != nil && !inputFormatFlag.Changed {
				rt.FormatInput = rt.Config.DefaultInputFormat
				cli.FormatInput = rt.Config.DefaultInputFormat // Keep global in sync
			}

			outputFormatFlag := cmd.Flag("format-output")
			if !useGlobalSync && rt.Config.DefaultOutputFormat != "" && outputFormatFlag != nil && !outputFormatFlag.Changed {
				rt.FormatOutput = rt.Config.DefaultOutputFormat
				// Don't update global for test-injected runtime
			} else if useGlobalSync && rt.Config.DefaultOutputFormat != "" && outputFormatFlag != nil && !outputFormatFlag.Changed {
				rt.FormatOutput = rt.Config.DefaultOutputFormat
				cli.FormatOutput = rt.Config.DefaultOutputFormat // Keep global in sync
			}

			// Store runtime in command context for child commands to access
			cmd.SetContext(rt.WithContext(cmd.Context()))

			return nil
		},
		// With no subcommand, print usage and exit successfully.
		RunE: cli.PrintUsage,
	}

	// Create root command flags
	// Note: Flags are bound to global variables for backward compatibility during transition
	rootCmd.PersistentFlags().StringVarP(&cli.ConfigFile, "config", "c", "", "path to configuration file to use")
	rootCmd.PersistentFlags().StringP("log-format", "L", "", "log format (json,rfc3339,basic)")
	rootCmd.PersistentFlags().StringP("log-level", "l", "", "set verbosity of logs (info,warning,debug)")
	rootCmd.PersistentFlags().String("log-color", "", "set coloring of logs (auto,on,off)")
	rootCmd.PersistentFlags().StringP("cluster", "C", "", "name of cluster whose config to use for this command")
	rootCmd.PersistentFlags().StringP("cluster-uri", "u", "", "base URI for OpenCHAMI services, excluding service base path (overrides cluster.uri in config file)")
	rootCmd.PersistentFlags().StringVar(&cli.CACertPath, "cacert", "", "path to root CA certificate in PEM format")
	rootCmd.PersistentFlags().StringVarP(&cli.Token, "token", "t", "", "access token to present for authentication")
	rootCmd.PersistentFlags().Bool("no-token", false, "do not check for or use an access token")
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
	if code := handleExecuteError(rootCmd, err); code != cli.CodeSuccess {
		os.Exit(code)
	}
}

// handleExecuteError centralizes the post-Execute error handling: it logs the
// failure, emits the appropriate "--help" hint for the invoked (or root)
// command, and returns the process exit code the error resolves to. It is
// separated from Execute so the logic can be exercised without terminating the
// test binary via os.Exit. A nil error yields CodeSuccess.
func handleExecuteError(rootCmd *cobra.Command, err error) int {
	if err == nil {
		return cli.CodeSuccess
	}
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
	return cli.ExitCode(err)
}
