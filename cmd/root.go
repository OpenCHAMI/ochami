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

// rootOptions holds the root-level flag values for this command tree.
// These are owned by the command tree, not global state.
type rootOptions struct {
	configFile   string
	cacertPath   string
	token        string
	insecure     bool
	logFormat    string
	logLevel     string
	logColor     string
	cluster      string
	clusterURI   string
	noToken      bool
	showToken    bool
	ignoreConfig bool
	verbose      bool
}

func NewRootCmd() *cobra.Command {
	// Create runtime instance for this command tree
	rt := cli.NewRuntime()

	// root-level options owned by this command tree
	var rootOpts rootOptions

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
			var err error
			rt, err = cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Only explicitly changed flags replace values supplied by an injected
			// runtime. This keeps config defaults and test fixtures intact.
			if cmd.Flag("config").Changed {
				rt.ConfigFile = rootOpts.configFile
			}
			if cmd.Flag("cacert").Changed {
				rt.CACertPath = rootOpts.cacertPath
			}
			if cmd.Flag("token").Changed {
				rt.Token = rootOpts.token
			}
			if cmd.Flag("insecure").Changed {
				rt.Insecure = rootOpts.insecure
			}
			rt.EarlyVerbose = rootOpts.verbose

			// Production runtimes load normal sources. Test runtimes load only an
			// explicitly selected source, including built-in defaults requested by
			// --ignore-config.
			loadConfig := rt.LoadConfig || cmd.Flag("config").Changed || cmd.Flag("ignore-config").Changed
			if commandEditsConfig(cmd) {
				loadConfig = false
				if rt.UserConfigFile == "" {
					if err := rt.ResolveUserConfigFile(); err != nil {
						return cli.Errorf(cli.CodeConfig, "failed to resolve user config path: %w", err)
					}
				}
			}
			if loadConfig {
				createConfig := commandMayCreateConfig(cmd)
				if err := rt.InitConfigAndLogging(cmd, createConfig); err != nil {
					return err
				}
			}

			// Config defaults apply only when the corresponding command flag was
			// not explicitly supplied.
			inputFormatFlag := cmd.Flag("format-input")
			if rt.Config.DefaultInputFormat != "" && (inputFormatFlag == nil || !inputFormatFlag.Changed) {
				rt.FormatInput = rt.Config.DefaultInputFormat
			}
			outputFormatFlag := cmd.Flag("format-output")
			if rt.Config.DefaultOutputFormat != "" && (outputFormatFlag == nil || !outputFormatFlag.Changed) {
				rt.FormatOutput = rt.Config.DefaultOutputFormat
			}
			if err := cli.ApplyFormatFlags(cmd, rt); err != nil {
				return err
			}

			return nil
		},
		// With no subcommand, print usage and exit successfully.
		RunE: cli.PrintUsage,
	}
	rootCmd.SetContext(rt.WithContext(rootCmd.Context()))

	// Create root command flags - all bound to command-tree-local rootOpts
	rootCmd.PersistentFlags().StringVarP(&rootOpts.configFile, "config", "c", "", "path to configuration file to use")
	rootCmd.PersistentFlags().StringVarP(&rootOpts.logFormat, "log-format", "L", "", "log format (json,rfc3339,basic)")
	rootCmd.PersistentFlags().StringVarP(&rootOpts.logLevel, "log-level", "l", "", "set verbosity of logs (info,warning,debug)")
	rootCmd.PersistentFlags().StringVar(&rootOpts.logColor, "log-color", "", "set coloring of logs (auto,on,off)")
	rootCmd.PersistentFlags().StringVarP(&rootOpts.cluster, "cluster", "C", "", "name of cluster whose config to use for this command")
	rootCmd.PersistentFlags().StringVarP(&rootOpts.clusterURI, "cluster-uri", "u", "", "base URI for OpenCHAMI services, excluding service base path (overrides cluster.uri in config file)")
	rootCmd.PersistentFlags().StringVar(&rootOpts.cacertPath, "cacert", "", "path to root CA certificate in PEM format")
	rootCmd.PersistentFlags().StringVarP(&rootOpts.token, "token", "t", "", "access token to present for authentication")
	rootCmd.PersistentFlags().BoolVar(&rootOpts.noToken, "no-token", false, "do not check for or use an access token")
	rootCmd.PersistentFlags().BoolVar(&rootOpts.showToken, "show-token", false, "show full access token in debug logs instead of a truncated value")
	rootCmd.PersistentFlags().BoolVarP(&rootOpts.insecure, "insecure", "k", false, "do not verify TLS certificates")
	rootCmd.PersistentFlags().BoolVar(&rootOpts.ignoreConfig, "ignore-config", false, "do not use any config file")
	rootCmd.PersistentFlags().BoolVarP(&rootOpts.verbose, "verbose", "v", false, "be verbose before logging is initialized")

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

func commandMayCreateConfig(cmd *cobra.Command) bool {
	switch cmd.CommandPath() {
	case "ochami config show", "ochami config unset", "ochami config cluster show", "ochami config cluster unset":
		return false
	default:
		return true
	}
}

func commandEditsConfig(cmd *cobra.Command) bool {
	switch cmd.CommandPath() {
	case "ochami config set", "ochami config unset", "ochami config cluster set", "ochami config cluster unset", "ochami config cluster delete":
		return true
	default:
		return false
	}
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
