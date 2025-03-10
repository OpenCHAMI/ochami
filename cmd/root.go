// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/OpenCHAMI/ochami/internal/config"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/internal/version"
	"github.com/spf13/cobra"
)

const (
	defaultInputFormat  = "json"
	defaultOutputFormat = "json"
)

var (
	configFile string
	logLevel   string
	logFormat  string

	// These are only used by subcommands.
	cacertPath string
	token      string
	insecure   bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   config.ProgName,
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
		//   initConfigAndLogging(cmd, false)
		//
		initConfigAndLogging(cmd, true)

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Logger.Error().Err(err).Msg("failed to execute command")
		if cmd, _, err := rootCmd.Find(os.Args[1:]); err != nil {
			// Error looking up invoked command, default to printing
			// help suggestion for root command, printing debug
			// message only for debugging (most users don't need to
			// know an error occurred).
			log.Logger.Debug().Err(err).Msg("failed to lookup invoked command")
			logHelpError(rootCmd)
		} else {
			// Print help suggestion for invoked command
			logHelpError(cmd)
		}
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "path to configuration file to use")
	rootCmd.PersistentFlags().StringP("log-format", "L", "", "log format (json,rfc3339,basic)")
	rootCmd.PersistentFlags().StringP("log-level", "l", "", "set verbosity of logs (info,warning,debug)")
	rootCmd.PersistentFlags().StringP("cluster", "C", "", "name of cluster whose config to use for this command")
	rootCmd.PersistentFlags().StringP("cluster-uri", "u", "", "base URI for OpenCHAMI services, excluding service base path (overrides cluster.uri in config file)")
	rootCmd.PersistentFlags().StringVar(&cacertPath, "cacert", "", "path to root CA certificate in PEM format")
	rootCmd.PersistentFlags().StringVarP(&token, "token", "t", "", "access token to present for authentication")
	rootCmd.PersistentFlags().BoolVarP(&insecure, "insecure", "k", false, "do not verify TLS certificates")
	rootCmd.PersistentFlags().Bool("ignore-config", false, "do not use any config file")
	rootCmd.PersistentFlags().BoolVarP(&config.EarlyVerbose, "verbose", "v", false, "be verbose before logging is initialized")

	// Either use cluster from config file or specify details on CLI
	rootCmd.MarkFlagsMutuallyExclusive("cluster", "cluster-uri")
}
