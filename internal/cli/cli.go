// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

// lib.go provides library functions to the cmd package, a.k.a. all cobra
// commands.

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/jwt"
	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/config"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/discover"
	"github.com/OpenCHAMI/ochami/pkg/format"

	"github.com/OpenCHAMI/ochami/internal/version"
)

var (
	// Errors
	FileExistsError   = fmt.Errorf("file exists")
	NoConfigFileError = fmt.Errorf("no config file to read")

	// el is an early logger that has verbosity turned on automatically.
	// It is for printing log messages before logging has been initialized,
	// regardless of --verbose.
	el = log.NewBasicLogger(os.Stderr, true, version.ProgName)

	// Standard ioStream that writes to the regular OS's input/output
	// streams.
	Ios = newIOStream(os.Stdin, os.Stdout, os.Stderr)

	// Global config file path (set externally by importer)
	ConfigFile string

	// Used by subcommands
	Token      string
	CACertPath string
	Insecure   bool

	// Variables to store values of --format-output and --format-input.
	// Default values are set here.
	FormatInput  = format.DataFormatJson
	FormatOutput = format.DataFormatJson
)

// ioStream provides a way to change the input and/or output stream for
// functions that read from os.Stdin and/or write to os.Stdout/os.Stderr. This
// is so that they can be more easily unit tested without having to modify
// os.Std*.
type ioStream struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func newIOStream(stdin io.Reader, stdout, stderr io.Writer) ioStream {
	return ioStream{
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	}
}

// AskToCreate prompts the user to, if path does not exist, to create a blank
// file at path. If it exists, nil is returned. If the user declines, a
// UserDeclinedError is returned. If an error occurs during creation, an error
// is returned.
func (i ioStream) AskToCreate(path string) (bool, error) {
	if path == "" {
		return false, fmt.Errorf("path cannot be empty")
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		respConfigCreate, err2 := i.LoopYesNo(fmt.Sprintf("%s does not exist. Create it?", path))
		if err2 != nil {
			return false, fmt.Errorf("error fetching user input: %w", err2)
		} else if respConfigCreate {
			return true, nil
		}
	} else {
		return false, FileExistsError
	}

	return false, nil
}

// LoopYesNo takes prompt p and appends " [yN]: " to it and prompts the user for
// input. As long as the user's input is not "y" or "n" (case insensitive), the
// function redisplays the prompt. If the user's response is "y", true is
// returned. If the user's response is "n", false is returned.
func (i ioStream) LoopYesNo(p string) (bool, error) {
	s := bufio.NewScanner(i.stdin)

	for {
		fmt.Fprint(i.stderr, fmt.Sprintf("%s [yn]:", p))
		if !s.Scan() {
			break
		}
		resp := strings.TrimSpace(s.Text())
		switch strings.ToLower(resp) {
		case "y":
			return true, nil
		case "n":
			return false, nil
		default:
			continue
		}
	}
	return false, s.Err()
}

// InitConfig initializes the global configuration for a command, creating the
// config file if create is true, if it does not already exist.
func InitConfig(cmd *cobra.Command, create bool) error {
	// Do not read or write config file if --ignore-config passed
	if cmd.Flags().Changed("ignore-config") {
		return nil
	}

	if ConfigFile != "" {
		if create {
			// Try to create config file with default values if it doesn't exist
			if cr, err := Ios.AskToCreate(ConfigFile); err != nil {
				// Only return error if error is not one that the file
				// already exists.
				if !errors.Is(err, FileExistsError) {
					// Error occurred during prompt
					return fmt.Errorf("error occurred asking to create config file: %w", err)
				}
			} else if cr {
				// User answered yes
				if err := CreateIfNotExists(ConfigFile); err != nil {
					return fmt.Errorf("failed to create %s: %w", ConfigFile, err)
				}
			} else {
				// User answered no
				return fmt.Errorf("user declined to create file; exiting...")
			}
		}
	}

	// Read configuration from file, if passed or merge config from system
	// config file and user config file if not passed.
	var err error
	if ConfigFile != "" {
		err = config.LoadGlobalConfigFromFile(ConfigFile)
	} else {
		err = config.LoadGlobalConfigMerged()
	}
	if err != nil {
		err = fmt.Errorf("failed to load configuration: %w", err)
	}

	return err
}

// Set log level verbosity based on config file (log.level) or --log-level.
// The command line option overrides the config file option.
func InitLogging(cmd *cobra.Command) error {
	if cmd.Flags().Changed("log-format") {
		lf, err := cmd.Flags().GetString("log-format")
		if err != nil {
			return fmt.Errorf("failed to fetch flag log-format: %w", err)
		}
		config.GlobalConfig.Log.Format = lf
	}
	if cmd.Flags().Changed("log-level") {
		ll, err := cmd.Flags().GetString("log-level")
		if err != nil {
			return fmt.Errorf("failed to fetch flag log-level: %w", err)
		}
		config.GlobalConfig.Log.Level = ll
	}

	if err := log.Init(config.GlobalConfig.Log.Level, config.GlobalConfig.Log.Format); err != nil {
		return fmt.Errorf("failed to Initialize logger: %w", err)
	}

	log.Logger.Debug().Msg("logging has been initialized")
	return nil
}

// InitConfigAndLogging is a wrapper around the config and logging init
// functions that is meant to be the first thing a command runs in its "Run"
// directive. createCfg determines whether a config file should be created if
// missing. This creation only applies when a config file is explicitly
// specified on the command line and not the merged config.
func InitConfigAndLogging(cmd *cobra.Command, createCfg bool) {
	if err := InitConfig(cmd, createCfg); err != nil {
		el.BasicLogf("failed to initialize config: %v", err)
		el.BasicLogf("see '%s --help' for long command help", cmd.CommandPath())
		os.Exit(1)
	}
	if err := InitLogging(cmd); err != nil {
		el.BasicLogf("failed to initialized logging: %v", err)
		el.BasicLogf("see '%s --help' for long command help", cmd.CommandPath())
		os.Exit(1)
	}
}

// CreateIfNotExists creates path (a file with optional leading directories) if
// any of the path components do not exist, returning an error if one occurred
// with the creation.
func CreateIfNotExists(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		parentDir := filepath.Dir(path)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("could not create parent dir %s: %w", parentDir, err)
		}
		f, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0644)
		if err != nil {
			return fmt.Errorf("creating %s failed: %w", path, err)
		}
		f.Close()
	}

	return nil
}

// CheckToken takes a pointer to a Cobra command and checks to see if --token
// was set. If not, an error is printed and the program exits.
func CheckToken(cmd *cobra.Command) {
	// TODO: Check token validity/expiration
	if Token == "" {
		log.Logger.Error().Msg("no token set")
		os.Exit(1)
	}

	// Try to parse token
	t, err := jwt.ParseString(Token, jwt.WithValidate(false))
	if err != nil {
		log.Logger.Error().Err(err).Msg("failed to parse token")
		os.Exit(1)
	}

	// Check expiration
	now := time.Now()
	exp := t.Expiration()
	if exp.Compare(now) < 0 {
		log.Logger.Error().Msgf("token is expired (expired %s ago at %s)",
			now.Sub(exp), exp.Local().Format(time.RFC1123))
		os.Exit(1)
	} else if exp.Sub(now).Minutes() <= 15 {
		log.Logger.Warn().Msgf("%s until token expires", exp.Sub(now))
	}

	// Validate not before (nbf), issued at (iat), and expiration (exp) fields
	err = jwt.Validate(t,
		jwt.WithValidator(jwt.IsNbfValid()),
		jwt.WithValidator(jwt.IsIssuedAtValid()),
		jwt.WithValidator(jwt.IsExpirationValid()),
	)
	if err != nil {
		log.Logger.Error().Err(err).Msg("token is invalid")
		os.Exit(1)
	}
}

// UseCACert takes a pointer to a client.OchamiClient and, if a path to a CA
// certificate has been set via --cacert, it configures it to use it. If an
// error occurs, a log is printed and the program exits.
func UseCACert(client *client.OchamiClient) {
	if CACertPath != "" {
		log.Logger.Debug().Msgf("Attempting to use CA certificate at %s", CACertPath)
		if err := client.UseCACert(CACertPath); err != nil {
			log.Logger.Error().Err(err).Msgf("failed to load CA certificate %s", CACertPath)
			os.Exit(1)
		}
	}
}

func GetBaseURIBSS(cmd *cobra.Command) (string, error) {
	return GetBaseURI(cmd, config.ServiceBSS)
}

func GetBaseURICloudInit(cmd *cobra.Command) (string, error) {
	return GetBaseURI(cmd, config.ServiceCloudInit)
}

func GetBaseURIPCS(cmd *cobra.Command) (string, error) {
	return GetBaseURI(cmd, config.ServicePCS)
}

func GetBaseURISMD(cmd *cobra.Command) (string, error) {
	return GetBaseURI(cmd, config.ServiceSMD)
}

func GetBaseURI(cmd *cobra.Command, serviceName config.ServiceName) (string, error) {
	// Precedence of getting base URI for requests (higher numbers override
	// all preceding numbers):
	//
	// 1. If "default-cluster" is set in config file (config file must be
	//    specified), use cluster identified by that name as source of info.
	// 2. If --cluster is set, search config file for matching name and read
	//    details from there.
	// 3. If flags corresponding to cluster info (e.g. --cluster-uri,
	//    --uri) are set, read details from them.
	var (
		clusterName   string
		clusterToUse  config.ConfigCluster
		clusterConfig config.ConfigClusterConfig
		clusterList   = config.GlobalConfig.Clusters
	)
	if config.GlobalConfig.DefaultCluster != "" {
		// 3. Check 'default-cluster'.
		clusterName = config.GlobalConfig.DefaultCluster
		clusterList = config.GlobalConfig.Clusters
		log.Logger.Debug().Msgf("using base URI from default cluster %s", clusterName)
		for _, c := range clusterList {
			if c.Name == clusterName {
				clusterToUse = c
				break
			}
		}
		if clusterToUse == (config.ConfigCluster{}) {
			return "", fmt.Errorf("default cluster %s not found", clusterName)
		}
		clusterConfig = clusterToUse.Cluster
	} else if cmd.Flag("cluster").Changed {
		// 2. Check --cluster (overrides "default-cluster").
		clusterName = cmd.Flag("cluster").Value.String()
		log.Logger.Debug().Msgf("reading URI from cluster %s passed from command line", clusterName)
		for _, c := range clusterList {
			if c.Name == clusterName {
				clusterToUse = c
				break
			}
		}
		if clusterToUse == (config.ConfigCluster{}) {
			return "", fmt.Errorf("cluster %s not found", clusterName)
		}

		clusterConfig = clusterToUse.Cluster
	}
	// 1. Check flags (--cluster-uri and/or --uri) and override any
	// previously-set values while leaving unspecified ones alone.
	if cmd.Flag("cluster-uri").Changed || (cmd.Flag("uri") != nil && cmd.Flag("uri").Changed) {
		log.Logger.Debug().Msg("using base URI passed on command line")
		ccc := config.ConfigClusterConfig{URI: cmd.Flag("cluster-uri").Value.String()}
		switch serviceName {
		case config.ServiceBSS:
			ccc.BSS.URI = cmd.Flag("uri").Value.String()
		case config.ServiceCloudInit:
			ccc.CloudInit.URI = cmd.Flag("uri").Value.String()
		case config.ServicePCS:
			ccc.PCS.URI = cmd.Flag("uri").Value.String()
		case config.ServiceSMD:
			ccc.SMD.URI = cmd.Flag("uri").Value.String()
		default:
			return "", fmt.Errorf("unknown service %q specified when generating base URI", serviceName)
		}
		clusterConfig = clusterConfig.MergeURIConfig(ccc)
	}

	baseURI, err := clusterConfig.GetServiceBaseURI(serviceName)
	if err != nil {
		if strings.TrimSpace(clusterName) != "" {
			err = fmt.Errorf("could not get %s base URI for cluster %s: %w", serviceName, clusterName, err)
		} else {
			err = fmt.Errorf("could not get %s base URI: %w", serviceName, err)
		}
	}

	return baseURI, err
}

// HandleToken is a wrapper function around code that reads, checks, and
// performs any other setup tasks for tokens. It is called by all commands that
// require a token.
func HandleToken(cmd *cobra.Command) {
	if cmd.Flag("no-token").Changed {
		// --no-token overrides any cluster settings
		log.Logger.Debug().Msg("--no-token passed, not reading or checking for token")
	} else {
		// Check if enable-auth is set for cluster and only read/check
		// token if true
		var clusterName string
		if cmd.Flag("cluster").Changed {
			// Use cluster passed via --cluster
			clusterName = cmd.Flag("cluster").Value.String()
		} else if config.GlobalConfig.DefaultCluster != "" {
			// Use default cluster
			clusterName = config.GlobalConfig.DefaultCluster
		}

		if clusterName != "" {
			if cl, err := config.GlobalConfig.GetCluster(clusterName); err != nil {
				if errors.Is(err, config.ErrUnknownCluster{}) {
					// Cluster was not found (this error
					// should be caught before this function, but
					// this check is here just in case),
					// skip token check
					log.Logger.Warn().Msgf("cluster %q not found, not checking token", clusterName)
				} else {
					// Other error occurred, fatal
					log.Logger.Error().Err(err).Msg("failed to get cluster")
					LogHelpError(cmd)
					os.Exit(1)
				}
			} else {
				// Cluster was found, use enable-auth value to
				// determine whether to read/check token
				if cl.Cluster.EnableAuth {
					log.Logger.Debug().Msgf("authentication enabled for cluster %s, reading and checking token", cl.Name)
					SetToken(cmd)
					CheckToken(cmd)
				} else {
					log.Logger.Debug().Msgf("authentication disabled for cluster %s, not reading or checking for token", cl.Name)
				}
			}
		}
	}
}

// SetToken sets the access token for a cobra command cmd. If --token
// was passed, that value is set as the access token. Otherwise, the token is
// read from an environment variable whose format is <CLUSTER>_ACCESS_TOKEN
// where <CLUSTER> is the name of the cluster, in upper case, being contacted.
// The value of <CLUSTER> is determined by taking the cluster name, passed
// either by --cluster or reading default-cluster from the config file (the
// former preceding the latter), replacing spaces and dashes (-) with
// underscores, and making the letters uppercase. If no config file is set or
// the environment variable is not set, an error is logged and the program
// exits.
func SetToken(cmd *cobra.Command) {
	var (
		clusterName string
		varPrefix   string
	)
	if cmd.Flag("token").Changed {
		Token = cmd.Flag("token").Value.String()
		log.Logger.Debug().Msg("--token passed, setting token to its value: " + Token)
		return
	}

	log.Logger.Debug().Msg("Determining token from environment variable based on cluster in config file")
	if cmd.Flag("cluster").Changed {
		clusterName = cmd.Flag("cluster").Value.String()
		log.Logger.Debug().Msg("--cluster specified: " + clusterName)
	} else if config.GlobalConfig.DefaultCluster != "" {
		clusterName = config.GlobalConfig.DefaultCluster
		log.Logger.Debug().Msg("--cluster not specified, using default-cluster: " + clusterName)
	} else {
		log.Logger.Error().Msg("No default-cluster specified and --token not passed")
		LogHelpError(cmd)
		os.Exit(1)
	}

	varPrefix = strings.ReplaceAll(clusterName, "-", "_")
	varPrefix = strings.ReplaceAll(varPrefix, " ", "_")

	envVarToRead := strings.ToUpper(varPrefix) + "_ACCESS_TOKEN"
	log.Logger.Debug().Msg("Reading token from environment variable: " + envVarToRead)
	if t, tokenSet := os.LookupEnv(envVarToRead); tokenSet {
		log.Logger.Debug().Msgf("Token found from environment variable: %s=%s", envVarToRead, t)
		Token = t
		return
	}

	log.Logger.Error().Msgf("Environment variable %s unset for reading token for cluster %q", envVarToRead, clusterName)
	os.Exit(1)
	LogHelpError(cmd)
}

// HandlePayload unmarshals raw data or data from a payload file into v for
// command cmd if --data and, optionally, --format-input, are passed.
func HandlePayload(cmd *cobra.Command, v any) {
	if cmd.Flag("data").Changed {
		data := cmd.Flag("data").Value.String()
		if err := client.ReadPayload(data, FormatInput, v); err != nil {
			log.Logger.Error().Err(err).Msg("unable to read payload data or file")
			LogHelpError(cmd)
			os.Exit(1)
		}
	}
}

// HandlePayloadStdin is similar to HandlePayload except the data is read from
// standard input.
func HandlePayloadStdin(cmd *cobra.Command, v any) {
	if err := client.ReadPayloadStdin(FormatInput, v); err != nil {
		log.Logger.Error().Err(err).Msg("error reading payload data from stdin")
		os.Exit(1)
	}
}

// PrintUsageHandleError is a simple wrapper around printing a command's usage
// that handles errors.
func PrintUsageHandleError(cmd *cobra.Command) {
	if err := cmd.Usage(); err != nil {
		log.Logger.Error().Err(err).Msg("failed to print usage")
		os.Exit(1)
	}
	LogHelpWarn(cmd)
}

// LogHelpError logs a message at error level telling the user to use the
// '--help' flag of the passed command to get more information on the command.
// The full command invocation without flags or arguments is printed in the
// message.
func LogHelpError(cmd *cobra.Command) {
	log.Logger.Error().Msgf("see '%s --help' for long command help", cmd.CommandPath())
}

// LogHelpWarn logs a message at warn level telling the user to use the '--help'
// flag of the passed command to get more information on the command.  The full
// command invocation without flags or arguments is printed in the message.
func LogHelpWarn(cmd *cobra.Command) {
	log.Logger.Warn().Msgf("see '%s --help' for long command help", cmd.CommandPath())
}

// CompletionFormatData is the cobra completion function for any flag that uses
// the format.DataFormat type.
func CompletionFormatData(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var helpSlice []string
	for k, v := range format.DataFormatHelp {
		helpSlice = append(helpSlice, fmt.Sprintf("%s\t%s", k, v))
	}
	return helpSlice, cobra.ShellCompDirectiveDefault
}

// CompletionDiscoveryVersion is the cobra completion function for the
// --discovery-version flag.
func CompletionDiscoveryVersion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var helpSlice []string
	for k, v := range discover.DiscoveryVersionHelp {
		helpSlice = append(helpSlice, fmt.Sprintf("%d\t%s", k, v))
	}
	return helpSlice, cobra.ShellCompDirectiveDefault
}
