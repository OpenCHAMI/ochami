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

	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/config"
	"github.com/openchami/ochami/pkg/discover"
	"github.com/openchami/ochami/pkg/format"

	"github.com/openchami/ochami/internal/version"
)

var (
	// Errors
	FileExistsError = fmt.Errorf("file exists")

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

// SetIOStream replaces the package-global Ios with one backed by the provided
// streams and returns a function that restores the previous Ios. It exists so
// that callers (notably tests) can redirect interactive input/output and
// capture prompt output without modifying os.Std*.
//
// Typical use:
//
//	restore := cli.SetIOStream(strings.NewReader("y\n"), &out, &out)
//	defer restore()
func SetIOStream(stdin io.Reader, stdout, stderr io.Writer) (restore func()) {
	prev := Ios
	Ios = newIOStream(stdin, stdout, stderr)
	return func() { Ios = prev }
}

// In returns the stream's input reader. Commands that read interactive or piped
// input should use this instead of os.Stdin so the source can be swapped via
// SetIOStream.
func (i ioStream) In() io.Reader { return i.stdin }

// Out returns the stream's output writer. Commands that stream output should
// use this instead of os.Stdout so it can be captured via SetIOStream.
func (i ioStream) Out() io.Writer { return i.stdout }

// Err returns the stream's error writer.
func (i ioStream) Err() io.Writer { return i.stderr }

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
	if f := cmd.Flag("ignore-config"); f != nil && f.Value.String() == "true" {
		err := loadDefaultConfig()
		if err != nil {
			return fmt.Errorf("unable to load default config: %w", err)
		}
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
		err = loadConfigFromFile(ConfigFile)
	} else {
		err = loadMergedConfig()
	}
	if err != nil {
		return err
	}

	return nil
}

// Set log level verbosity based on config file (log.level) or --log-level.
// The command line option overrides the config file option.
func InitLogging(cmd *cobra.Command) error {
	// 1. Apply command-line overrides first (highest precedence)
	if cmd.Flags().Changed("log-format") {
		lf, err := cmd.Flags().GetString("log-format")
		if err != nil {
			return fmt.Errorf("failed to fetch flag log-format: %w", err)
		}
		activeConfig.Log.Format = lf
	}
	if cmd.Flags().Changed("log-level") {
		ll, err := cmd.Flags().GetString("log-level")
		if err != nil {
			return fmt.Errorf("failed to fetch flag log-level: %w", err)
		}
		activeConfig.Log.Level = ll
	}
	if cmd.Flags().Changed("log-color") {
		lc, err := cmd.Flags().GetString("log-color")
		if err != nil {
			return fmt.Errorf("failed to fetch flag log-color: %w", err)
		}
		activeConfig.Log.Color = lc
	}

	// 2. Apply defaults for empty values (lowest precedence)
	defaults := config.DefaultGlobalMap()
	if activeConfig.Log.Level == "" {
		activeConfig.Log.Level = defaults["log.level"].(string)
	}
	if activeConfig.Log.Format == "" {
		activeConfig.Log.Format = defaults["log.format"].(string)
	}
	if activeConfig.Log.Color == "" {
		activeConfig.Log.Color = defaults["log.color"].(string)
	}

	// 3. Initialize logger
	if err := log.Init(activeConfig.Log.Level, activeConfig.Log.Format, activeConfig.Log.Color); err != nil {
		return err
	}

	log.Logger.Debug().Msg("logging has been initialized")
	return nil
}

// InitConfigAndLogging is a wrapper around the config and logging init
// functions that is meant to be the first thing a command runs in its "Run"
// directive. createCfg determines whether a config file should be created if
// missing. This creation only applies when a config file is explicitly
// specified on the command line and not the merged config.
func InitConfigAndLogging(cmd *cobra.Command, createCfg bool) error {
	// Load configuration first (this populates GlobalConfig)
	if err := InitConfig(cmd, createCfg); err != nil {
		return Errorf(CodeConfig, "failed to initialize config: %w", err)
	}
	// Initialize logging second (flag overrides are applied inside InitLogging)
	if err := InitLogging(cmd); err != nil {
		return Errorf(CodeConfig, "failed to initialize logging: %w", err)
	}
	return nil
}

// ShowToken reports whether the --show-token flag was passed for cmd, indicating
// that full access tokens should be shown in debug logs instead of being
// truncated. It returns false if the flag is not defined for the command.
func ShowToken(cmd *cobra.Command) bool {
	if f := cmd.Flag("show-token"); f != nil {
		return f.Value.String() == "true"
	}
	return false
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
		if err := f.Close(); err != nil {
			return fmt.Errorf("closing %s failed: %w", path, err)
		}
	}

	return nil
}

// CheckToken takes a pointer to a Cobra command and checks to see if --token
// was set. If not, or if the token is invalid or expired, a CodeAuth CodedError
// is returned.
func CheckToken(cmd *cobra.Command) error {
	if Token == "" {
		return Errorf(CodeAuth, "no token set")
	}

	// Parse and validate token (jwt.Parse validates nbf, iat, exp automatically
	// in v3). WithVerify(false) is used because the signature is not verified
	// here. Only the token's time-based claims (exp, nbf, iat) are checked.
	// Signature verification would be done by the services themselves.
	t, err := jwt.Parse([]byte(Token), jwt.WithVerify(false))
	if err != nil {
		// Provide specific error messages based on error type
		if errors.Is(err, jwt.TokenExpiredError()) {
			return Errorf(CodeAuth, "token is expired")
		} else if errors.Is(err, jwt.TokenNotYetValidError()) {
			return Errorf(CodeAuth, "token is not yet valid (nbf in future)")
		} else if errors.Is(err, jwt.InvalidIssuerError()) {
			return Errorf(CodeAuth, "token has invalid issuer")
		} else if errors.Is(err, jwt.InvalidAudienceError()) {
			return Errorf(CodeAuth, "token has invalid audience")
		}
		return Errorf(CodeAuth, "failed to parse token: %w", err)
	}

	// Manual expiration check for "expiring soon" warning. jwt.Parse() already
	// validated exp/nbf/iat, so this is just for the warning.
	now := time.Now()
	exp, ok := t.Expiration()
	if ok && exp.Sub(now).Minutes() <= 15 && exp.After(now) {
		log.Logger.Warn().Msgf("%s until token expires", exp.Sub(now))
	}

	return nil
}

// UseCACert takes a pointer to a client.OchamiClient and, if a path to a CA
// certificate has been set via --cacert, it configures it to use it. If an
// error occurs loading the certificate, a CodePayload CodedError is returned.
func UseCACert(client *client.OchamiClient) error {
	if CACertPath != "" {
		log.Logger.Debug().Msgf("Attempting to use CA certificate at %s", CACertPath)
		if err := client.UseCACert(CACertPath); err != nil {
			return Errorf(CodePayload, "failed to load CA certificate %s: %w", CACertPath, err)
		}
	}
	return nil
}

func GetBaseURIMetadataService(cmd *cobra.Command) (string, error) {
	return GetBaseURI(cmd, config.ServiceMetadata)
}

func GetBaseURIBootService(cmd *cobra.Command) (string, error) {
	return GetBaseURI(cmd, config.ServiceBoot)
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

func GetBaseURIRCS(cmd *cobra.Command) (string, error) {
	return GetBaseURI(cmd, config.ServiceRCS)
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
		clusterList   = activeConfig.Clusters
	)
	if cmd.Flag("cluster").Changed {
		// An explicit cluster overrides the configured default cluster.
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
	} else if activeConfig.DefaultCluster != "" {
		// Check 'default-cluster' when --cluster was not passed.
		clusterName = activeConfig.DefaultCluster
		clusterList = activeConfig.Clusters
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
	}
	// Check flags (--cluster-uri and/or --uri) and override any
	// previously-set values while leaving unspecified ones alone.
	if cmd.Flag("cluster-uri").Changed || (cmd.Flag("uri") != nil && cmd.Flag("uri").Changed) {
		log.Logger.Debug().Msg("using base URI passed on command line")
		var ccc config.ConfigClusterConfig
		if cmd.Flag("cluster-uri").Changed {
			ccc.URI = cmd.Flag("cluster-uri").Value.String()
		}
		if cmd.Flag("uri") != nil && cmd.Flag("uri").Changed {
			switch serviceName {
			case config.ServiceBoot:
				ccc.BootService.URI = cmd.Flag("uri").Value.String()
			case config.ServiceBSS:
				ccc.BSS.URI = cmd.Flag("uri").Value.String()
			case config.ServiceCloudInit:
				ccc.CloudInit.URI = cmd.Flag("uri").Value.String()
			case config.ServiceMetadata:
				ccc.MetadataService.URI = cmd.Flag("uri").Value.String()
			case config.ServicePCS:
				ccc.PCS.URI = cmd.Flag("uri").Value.String()
			case config.ServiceSMD:
				ccc.SMD.URI = cmd.Flag("uri").Value.String()
			case config.ServiceRCS:
				ccc.RCS.URI = cmd.Flag("uri").Value.String()
			default:
				return "", fmt.Errorf("unknown service %q specified when generating base URI", serviceName)
			}
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

func GetAPIVersion(cmd *cobra.Command, serviceName config.ServiceName) (string, error) {
	// Precedence of getting API version for requests (higher numbers override
	// all preceding numbers):
	//
	// 1. If "default-cluster" is set in config file (config file must be
	//    specified), use cluster identified by that name as source of info.
	// 2. If --cluster is set, search config file for matching name and read
	//    details from there.
	// 3. If flags corresponding to cluster info (e.g. --cluster-uri,
	//    --uri) are set, read details from them.
	var (
		apiVersion    string
		clusterName   string
		clusterToUse  config.ConfigCluster
		clusterConfig config.ConfigClusterConfig
		clusterList   = activeConfig.Clusters
	)
	if cmd.Flag("cluster").Changed {
		// An explicit cluster overrides the configured default cluster.
		clusterName = cmd.Flag("cluster").Value.String()
		log.Logger.Debug().Msgf("reading API version for %s from cluster %s passed from command line", serviceName, clusterName)
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
	} else if activeConfig.DefaultCluster != "" {
		// Check 'default-cluster' when --cluster was not passed.
		clusterName = activeConfig.DefaultCluster
		clusterList = activeConfig.Clusters
		log.Logger.Debug().Msgf("using API version from %s in default cluster %s", serviceName, clusterName)
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
	}

	if !cmd.Flag("api-version").Changed {
		switch serviceName {
		case config.ServiceBoot:
			apiVersion = clusterConfig.BootService.APIVersion
		case config.ServiceMetadata:
			apiVersion = clusterConfig.MetadataService.APIVersion
		default:
			return "", fmt.Errorf("unknown service %q specified when fetching API version", serviceName)
		}
	} else {
		// Check flag (--api-version) and override any previously-set values
		// while leaving unspecified ones alone.
		apiVersion = cmd.Flag("api-version").Value.String()
	}

	return apiVersion, nil
}

// GetTimeout returns the timeout specified by --timeout, if passed. Otherwise,
// the config value of timeout is used. If that is not set, the compile-time
// default is used.
func GetTimeout(cmd *cobra.Command) time.Duration {
	if cmd.Flag("timeout").Changed {
		if dur, err := cmd.Flags().GetDuration("timeout"); err != nil {
			log.Logger.Warn().Err(err).Msgf("failed to get timeout from flag, falling back to config value of %s", activeConfig.Timeout)
		} else {
			return dur
		}
	}
	return activeConfig.Timeout
}

// HandleToken is a wrapper function around code that reads, checks, and
// performs any other setup tasks for tokens. It is called by all commands that
// require a token.
func HandleToken(cmd *cobra.Command) error {
	if f := cmd.Flag("no-token"); f != nil && f.Value.String() == "true" {
		// --no-token overrides any cluster settings
		log.Logger.Debug().Msg("--no-token passed, not reading or checking for token")
	} else {
		// Check if enable-auth is set for cluster and only read/check
		// token if true
		var clusterName string
		if cmd.Flag("cluster").Changed {
			// Use cluster passed via --cluster
			clusterName = cmd.Flag("cluster").Value.String()
		} else if activeConfig.DefaultCluster != "" {
			// Use default cluster
			clusterName = activeConfig.DefaultCluster
		}

		if clusterName != "" {
			if cl, err := activeConfig.GetCluster(clusterName); err != nil {
				if errors.Is(err, config.ErrUnknownCluster{}) {
					// Cluster was not found (this error
					// should be caught before this function, but
					// this check is here just in case),
					// skip token check
					log.Logger.Warn().Msgf("cluster %q not found, not checking token", clusterName)
				} else {
					// Other error occurred, fatal
					return Errorf(CodeConfig, "failed to get cluster: %w", err)
				}
			} else {
				// Cluster was found, use enable-auth value to
				// determine whether to read/check token
				if cl.Cluster.EnableAuth {
					log.Logger.Debug().Msgf("authentication enabled for cluster %s, reading and checking token", cl.Name)
					if err := SetToken(cmd); err != nil {
						return err
					}
					if err := CheckToken(cmd); err != nil {
						return err
					}
				} else {
					log.Logger.Debug().Msgf("authentication disabled for cluster %s, not reading or checking for token", cl.Name)
				}
			}
		}
	}
	return nil
}

// SetToken sets the access token for a cobra command cmd. If --token
// was passed, that value is set as the access token. Otherwise, the token is
// read from an environment variable whose format is <CLUSTER>_ACCESS_TOKEN
// where <CLUSTER> is the name of the cluster, in upper case, being contacted.
// The value of <CLUSTER> is determined by taking the cluster name, passed
// either by --cluster or reading default-cluster from the config file (the
// former preceding the latter), replacing spaces and dashes (-) with
// underscores, and making the letters uppercase. If no config file is set or
// the environment variable is not set, a CodeAuth CodedError is returned.
func SetToken(cmd *cobra.Command) error {
	var (
		clusterName string
		varPrefix   string
	)
	if cmd.Flag("token").Changed {
		Token = cmd.Flag("token").Value.String()
		log.Logger.Debug().Msg("--token passed, setting token to its value: " + client.RedactToken(Token, ShowToken(cmd)))
		return nil
	}

	log.Logger.Debug().Msg("Determining token from environment variable based on cluster in config file")
	if cmd.Flag("cluster").Changed {
		clusterName = cmd.Flag("cluster").Value.String()
		log.Logger.Debug().Msg("--cluster specified: " + clusterName)
	} else if activeConfig.DefaultCluster != "" {
		clusterName = activeConfig.DefaultCluster
		log.Logger.Debug().Msg("--cluster not specified, using default-cluster: " + clusterName)
	} else {
		return Errorf(CodeAuth, "no default-cluster specified and --token not passed")
	}

	varPrefix = strings.ReplaceAll(clusterName, "-", "_")
	varPrefix = strings.ReplaceAll(varPrefix, " ", "_")

	envVarToRead := strings.ToUpper(varPrefix) + "_ACCESS_TOKEN"
	log.Logger.Debug().Msg("Reading token from environment variable: " + envVarToRead)
	if t, tokenSet := os.LookupEnv(envVarToRead); tokenSet {
		log.Logger.Debug().Msgf("Token found from environment variable: %s=%s", envVarToRead, client.RedactToken(t, ShowToken(cmd)))
		Token = t
		return nil
	}

	return Errorf(CodeAuth, "environment variable %s unset for reading token for cluster %q", envVarToRead, clusterName)
}

// HandlePayload unmarshals raw data or data from a payload file into v for
// command cmd if --data and, optionally, --format-input, are passed.
func HandlePayload(cmd *cobra.Command, v any) error {
	if cmd.Flag("data").Changed {
		data := cmd.Flag("data").Value.String()
		if err := client.ReadPayload(data, FormatInput, v); err != nil {
			return Errorf(CodePayload, "unable to read payload data or file: %w", err)
		}
	}
	return nil
}

// HandlePayloadSlice is similar to HandlePayload except that it unmarshals the
// payload data into a typed slice.
func HandlePayloadSlice[T any](cmd *cobra.Command, v *[]T) error {
	if cmd.Flag("data").Changed {
		data := cmd.Flag("data").Value.String()
		if err := client.ReadPayloadSlice[T](data, FormatInput, v); err != nil {
			return Errorf(CodePayload, "unable to read payload data or file into slice: %w", err)
		}
	}
	return nil
}

// HandlePayloadStdin is similar to HandlePayload except the data is read from
// standard input.
func HandlePayloadStdin(cmd *cobra.Command, v any) error {
	if err := client.ReadPayloadReader(Ios.In(), FormatInput, v); err != nil {
		return Errorf(CodePayload, "error reading payload data from stdin: %w", err)
	}
	return nil
}

// HandlePayloadStdinSlice is similar to HandlePayloadStdin except that it
// unmarshals the payload data into a typed slice.
func HandlePayloadStdinSlice[T any](cmd *cobra.Command, v *[]T) error {
	if err := client.ReadPayloadReaderSlice[T](Ios.In(), FormatInput, v); err != nil {
		return Errorf(CodePayload, "error reading payload data from stdin: %w", err)
	}
	return nil
}

// PrintUsageHandleError prints a command's usage, returning an error (rather
// than exiting) if usage printing fails. It is used by metacommands that have
// no action of their own and simply display usage.
func PrintUsageHandleError(cmd *cobra.Command) error {
	if err := cmd.Usage(); err != nil {
		return Errorf(CodeGeneric, "failed to print usage: %w", err)
	}
	return nil
}

// PrintUsage is a Cobra RunE adapter for metacommands whose only direct action
// is to display their usage.
func PrintUsage(cmd *cobra.Command, args []string) error {
	return PrintUsageHandleError(cmd)
}

// logHelpHint logs a message at error level telling the user to use the
// '--help' flag of the passed command to get more information on the command.
// The full command invocation without flags or arguments is printed in the
// message. It is emitted centrally by Execute after a command fails.
func logHelpHint(cmd *cobra.Command) {
	log.Logger.Error().Msgf("see '%s --help' for long command help", cmd.CommandPath())
}

// LogHelpHint emits the "see '<cmd> --help'" hint for cmd. It is intended to be
// called centrally (by Execute) after a command has failed, preserving the
// prior behavior of pointing users at command help on error.
func LogHelpHint(cmd *cobra.Command) {
	logHelpHint(cmd)
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

// CompletionPatchMethod is the cobra completion function for the --patch-method
// flag.
func CompletionPatchMethod(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var helpSlice []string
	for k, v := range client.PatchMethodHelp {
		helpSlice = append(helpSlice, fmt.Sprintf("%s\t%s", k, v))
	}
	return helpSlice, cobra.ShellCompDirectiveDefault
}
