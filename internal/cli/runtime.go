// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/knadh/koanf/v2"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/configfile"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/config"
	"github.com/openchami/ochami/pkg/format"
)

// Runtime holds the invocation-owned state for the CLI. It replaces the
// previous global variables (Ios, FormatInput, FormatOutput, Token, ConfigFile,
// activeConfig, activeKoanf) to enable test isolation and parallel test execution.
type Runtime struct {
	// I/O streams
	Ios *IOStreams

	// Format settings
	FormatInput  format.DataFormat
	FormatOutput format.DataFormat

	// Authentication
	Token      string
	CACertPath string
	Insecure   bool

	// Configuration
	ConfigFile     string
	UserConfigFile string
	Config         config.Config
	Koanf          *koanf.Koanf

	// LoadConfig controls whether root command execution reads configuration
	// sources automatically. Production runtimes enable it; test runtimes leave
	// it disabled so tests never inspect host configuration accidentally.
	LoadConfig bool

	// EarlyVerbose enables configuration diagnostics before normal logging is
	// initialized. It is invocation-owned rather than bound to the global logger.
	EarlyVerbose bool
}

// IOStreams provides access to the runtime's I/O streams.
type IOStreams struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

// NewIOStreams creates a new IOStreams instance with the provided streams.
func NewIOStreams(stdin io.Reader, stdout, stderr io.Writer) *IOStreams {
	return &IOStreams{
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	}
}

// In returns the input reader.
func (i *IOStreams) In() io.Reader { return i.stdin }

// Out returns the output writer.
func (i *IOStreams) Out() io.Writer { return i.stdout }

// Err returns the error writer.
func (i *IOStreams) Err() io.Writer { return i.stderr }

// AskToCreate prompts the user to, if path does not exist, to create a blank
// file at path. If it exists, nil is returned. If the user declines, a
// FileExistsError is returned. If an error occurs during creation, an error
// is returned.
func (i *IOStreams) AskToCreate(path string) (bool, error) {
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
func (i *IOStreams) LoopYesNo(p string) (bool, error) {
	s := bufio.NewScanner(i.stdin)

	for {
		if _, err := fmt.Fprintf(i.stderr, "%s [yn]:", p); err != nil {
			return false, fmt.Errorf("failed to write prompt: %w", err)
		}
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

// WriteOutput writes all of data to w. Short writes and writer failures are
// returned as payload errors because successful CLI execution requires output
// delivery to complete.
func WriteOutput(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if err != nil {
			return Errorf(CodePayload, "failed to write command output: %w", err)
		}
		if n <= 0 || n > len(data) {
			return Errorf(CodePayload, "failed to write command output: %w", io.ErrShortWrite)
		}
		data = data[n:]
	}
	return nil
}

// WriteString writes all of s to w with the same error contract as WriteOutput.
func WriteString(w io.Writer, s string) error {
	return WriteOutput(w, []byte(s))
}

// NewRuntime creates a new Runtime instance for production use.
// It initializes with the current global IOStreams and default formats.
// This ensures that if SetIOStream() has been called (e.g., by tests),
// the runtime will use those streams instead of os.Stdin/Stdout/Stderr.
func NewRuntime() *Runtime {
	return &Runtime{
		Ios:          NewIOStreams(os.Stdin, os.Stdout, os.Stderr),
		FormatInput:  format.DataFormatJson,
		FormatOutput: format.DataFormatJson,
		LoadConfig:   true,
	}
}

// NewTestRuntime creates a new Runtime instance for testing.
// It accepts custom I/O streams and allows setting formats, token, and config explicitly.
func NewTestRuntime(stdin io.Reader, stdout, stderr io.Writer) *Runtime {
	return &Runtime{
		Ios:          NewIOStreams(stdin, stdout, stderr),
		FormatInput:  format.DataFormatJson,
		FormatOutput: format.DataFormatJson,
	}
}

// WithConfigFile sets the config file path for the runtime.
func (rt *Runtime) WithConfigFile(path string) *Runtime {
	rt.ConfigFile = path
	return rt
}

// WithToken sets the authentication token for the runtime.
func (rt *Runtime) WithToken(token string) *Runtime {
	rt.Token = token
	return rt
}

// WithFormats sets the input and output formats for the runtime.
func (rt *Runtime) WithFormats(input, output format.DataFormat) *Runtime {
	rt.FormatInput = input
	rt.FormatOutput = output
	return rt
}

// WithCACert sets the CA certificate path for the runtime.
func (rt *Runtime) WithCACert(path string) *Runtime {
	rt.CACertPath = path
	return rt
}

// WithInsecure sets the insecure flag for the runtime.
func (rt *Runtime) WithInsecure(insecure bool) *Runtime {
	rt.Insecure = insecure
	return rt
}

// WithConfig sets the configuration for the runtime.
func (rt *Runtime) WithConfig(cfg config.Config) *Runtime {
	rt.Config = cfg
	return rt
}

// WithKoanf sets the koanf instance for the runtime.
func (rt *Runtime) WithKoanf(k *koanf.Koanf) *Runtime {
	rt.Koanf = k
	return rt
}

// WithIOStreams sets the I/O streams for the runtime.
func (rt *Runtime) WithIOStreams(stdin io.Reader, stdout, stderr io.Writer) *Runtime {
	rt.Ios = NewIOStreams(stdin, stdout, stderr)
	return rt
}

// ResolveUserConfigFile records the platform-specific user configuration path
// without reading that file.
func (rt *Runtime) ResolveUserConfigFile() error {
	path, err := config.UserConfigPath()
	if err != nil {
		return err
	}
	rt.UserConfigFile = path
	return nil
}

// runtimeKey is the context key for storing Runtime instances.
type runtimeKey struct{}

// WithContext stores the runtime in the provided context and returns the new context.
func (rt *Runtime) WithContext(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, runtimeKey{}, rt)
}

// FromContext retrieves the Runtime from the context. Returns nil, false if not found.
func FromContext(ctx context.Context) (*Runtime, bool) {
	if ctx == nil {
		return nil, false
	}
	rt, ok := ctx.Value(runtimeKey{}).(*Runtime)
	return rt, ok && rt != nil
}

// RuntimeFromCommand retrieves the invocation runtime for cmd. Commands may be
// constructed independently for tests and integrations, so absence of root
// initialization is a configuration error rather than a panic.
func RuntimeFromCommand(cmd *cobra.Command) (*Runtime, error) {
	if cmd == nil {
		return nil, Errorf(CodeConfig, "CLI runtime is unavailable: command is nil")
	}
	rt, ok := FromContext(cmd.Context())
	if !ok {
		return nil, Errorf(CodeConfig, "CLI runtime is unavailable; execute commands through the root command")
	}
	if err := ApplyFormatFlags(cmd, rt); err != nil {
		return nil, err
	}
	return rt, nil
}

// ShowToken reports whether the --show-token flag was passed for cmd, indicating
// that full access tokens should be shown in debug logs instead of being
// truncated. It returns false if the flag is not defined for the command.
func (rt *Runtime) ShowToken(cmd *cobra.Command) bool {
	if f := cmd.Flag("show-token"); f != nil {
		return f.Value.String() == "true"
	}
	return false
}

// CheckToken validates the current token in the runtime.
func (rt *Runtime) CheckToken() error {
	if rt.Token == "" {
		return Errorf(CodeAuth, "no token set")
	}

	// Parse and validate token (jwt.Parse validates nbf, iat, exp automatically
	// in v3). WithVerify(false) is used because the signature is not verified
	// here. Only the token's time-based claims (exp, nbf, iat) are checked.
	// Signature verification would be done by the services themselves.
	t, err := jwt.Parse([]byte(rt.Token), jwt.WithVerify(false))
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

// SetTokenFromFlag sets the access token from the --token flag.
func (rt *Runtime) SetTokenFromFlag(cmd *cobra.Command) error {
	if cmd.Flag("token").Changed {
		rt.Token = cmd.Flag("token").Value.String()
		log.Logger.Debug().Msg("--token passed, setting token to its value: " + client.RedactToken(rt.Token, rt.ShowToken(cmd)))
		return nil
	}
	return nil
}

// SetTokenFromEnv sets the access token from environment variable based on cluster name.
func (rt *Runtime) SetTokenFromEnv(cmd *cobra.Command) error {
	var clusterName string
	if cmd.Flag("cluster").Changed {
		clusterName = cmd.Flag("cluster").Value.String()
		log.Logger.Debug().Msg("--cluster specified: " + clusterName)
	} else if rt.Config.DefaultCluster != "" {
		clusterName = rt.Config.DefaultCluster
		log.Logger.Debug().Msg("--cluster not specified, using default-cluster: " + clusterName)
	} else {
		return Errorf(CodeAuth, "no default-cluster specified and --token not passed")
	}

	varPrefix := strings.ReplaceAll(clusterName, "-", "_")
	varPrefix = strings.ReplaceAll(varPrefix, " ", "_")

	envVarToRead := strings.ToUpper(varPrefix) + "_ACCESS_TOKEN"
	log.Logger.Debug().Msg("Reading token from environment variable: " + envVarToRead)
	if t, tokenSet := os.LookupEnv(envVarToRead); tokenSet {
		log.Logger.Debug().Msgf("Token found from environment variable: %s=%s", envVarToRead, client.RedactToken(t, rt.ShowToken(cmd)))
		rt.Token = t
		return nil
	}

	return Errorf(CodeAuth, "environment variable %s unset for reading token for cluster %q", envVarToRead, clusterName)
}

// HandleToken handles token reading and validation using runtime state.
func (rt *Runtime) HandleToken(cmd *cobra.Command) error {
	if f := cmd.Flag("no-token"); f != nil && f.Value.String() == "true" {
		// --no-token overrides any cluster settings
		log.Logger.Debug().Msg("--no-token passed, not reading or checking for token")
		return nil
	}

	// Check if enable-auth is set for cluster and only read/check
	// token if true
	var clusterName string
	if cmd.Flag("cluster").Changed {
		// Use cluster passed via --cluster
		clusterName = cmd.Flag("cluster").Value.String()
	} else if rt.Config.DefaultCluster != "" {
		// Use default cluster
		clusterName = rt.Config.DefaultCluster
	}

	if clusterName != "" {
		cl, err := rt.Config.GetCluster(clusterName)
		if err != nil {
			return Errorf(CodeConfig, "failed to get cluster: %w", err)
		} else {
			// Cluster was found, use enable-auth value to
			// determine whether to read/check token
			if cl.Cluster.EnableAuth {
				log.Logger.Debug().Msgf("authentication enabled for cluster %s, reading and checking token", cl.Name)
				if err := rt.SetTokenFromFlag(cmd); err != nil {
					return err
				}
				if rt.Token == "" {
					if err := rt.SetTokenFromEnv(cmd); err != nil {
						return err
					}
				}
				if err := rt.CheckToken(); err != nil {
					return err
				}
			} else {
				log.Logger.Debug().Msgf("authentication disabled for cluster %s, not reading or checking for token", cl.Name)
			}
		}
	}
	return nil
}

// UseCACert configures client with CA certificate from runtime.
func (rt *Runtime) UseCACert(ochamiClient *client.OchamiClient) error {
	if rt.CACertPath != "" {
		log.Logger.Debug().Msgf("Attempting to use CA certificate at %s", rt.CACertPath)
		if err := ochamiClient.UseCACert(rt.CACertPath); err != nil {
			return Errorf(CodePayload, "failed to load CA certificate %s: %w", rt.CACertPath, err)
		}
	}
	return nil
}

// GetTimeout returns the timeout from flag or config.
func (rt *Runtime) GetTimeout(cmd *cobra.Command) time.Duration {
	if cmd.Flag("timeout").Changed {
		if dur, err := cmd.Flags().GetDuration("timeout"); err == nil {
			return dur
		} else {
			log.Logger.Warn().Err(err).Msgf("failed to get timeout from flag, falling back to config value of %s", rt.Config.Timeout)
		}
	}
	return rt.Config.Timeout
}

// HandlePayload unmarshals raw data or data from a payload file into v for
// command cmd if --data and, optionally, --format-input, are passed.
func (rt *Runtime) HandlePayload(cmd *cobra.Command, v any) error {
	if cmd.Flag("data").Changed {
		data := cmd.Flag("data").Value.String()
		if err := client.ReadPayload(data, rt.FormatInput, v); err != nil {
			return Errorf(CodePayload, "unable to read payload data or file: %w", err)
		}
	}
	return nil
}

// HandlePayloadStdin is similar to HandlePayload except the data is read from
// standard input.
func (rt *Runtime) HandlePayloadStdin(cmd *cobra.Command, v any) error {
	if err := client.ReadPayloadReader(rt.Ios.In(), rt.FormatInput, v); err != nil {
		return Errorf(CodePayload, "error reading payload data from stdin: %w", err)
	}
	return nil
}

// HandlePayloadSliceWithRuntime is similar to Runtime.HandlePayload except that
// it unmarshals the payload data into a typed slice. It is a package function
// (rather than a method) because Go does not permit type parameters on methods.
func HandlePayloadSliceWithRuntime[T any](rt *Runtime, cmd *cobra.Command, v *[]T) error {
	if cmd.Flag("data").Changed {
		data := cmd.Flag("data").Value.String()
		if err := client.ReadPayloadSlice[T](data, rt.FormatInput, v); err != nil {
			return Errorf(CodePayload, "unable to read payload data or file into slice: %w", err)
		}
	}
	return nil
}

// HandlePayloadStdinSliceWithRuntime is similar to Runtime.HandlePayloadStdin
// except that it unmarshals the payload data into a typed slice. It is a package
// function (rather than a method) because Go does not permit type parameters on
// methods.
func HandlePayloadStdinSliceWithRuntime[T any](rt *Runtime, cmd *cobra.Command, v *[]T) error {
	if err := client.ReadPayloadReaderSlice[T](rt.Ios.In(), rt.FormatInput, v); err != nil {
		return Errorf(CodePayload, "error reading payload data from stdin: %w", err)
	}
	return nil
}

// GetBaseURI returns base URI for a service.
func (rt *Runtime) GetBaseURI(cmd *cobra.Command, serviceName config.ServiceName) (string, error) {
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
		clusterList   = rt.Config.Clusters
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
	} else if rt.Config.DefaultCluster != "" {
		// Check 'default-cluster' when --cluster was not passed.
		clusterName = rt.Config.DefaultCluster
		clusterList = rt.Config.Clusters
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

// GetAPIVersion returns API version for a service.
func (rt *Runtime) GetAPIVersion(cmd *cobra.Command, serviceName config.ServiceName) (string, error) {
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
		clusterList   = rt.Config.Clusters
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
	} else if rt.Config.DefaultCluster != "" {
		// Check 'default-cluster' when --cluster was not passed.
		clusterName = rt.Config.DefaultCluster
		clusterList = rt.Config.Clusters
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

// InitConfig initializes runtime configuration from command.
func (rt *Runtime) InitConfig(cmd *cobra.Command, create bool) error {
	logger := earlyLogger{logger: log.NewBasicLogger(rt.Ios.Err(), rt.EarlyVerbose, "ochami")}

	// Do not read or write config file if --ignore-config passed
	if f := cmd.Flag("ignore-config"); f != nil && f.Value.String() == "true" {
		cfg, err := config.LoadDefaults(config.WithLogger(logger))
		if err != nil {
			return fmt.Errorf("unable to load default config: %w", err)
		}
		rt.Config = cfg
		ko, err := configfile.EffectiveKoanf()
		if err != nil {
			return fmt.Errorf("unable to load default config representation: %w", err)
		}
		rt.Koanf = ko
		return nil
	}

	if rt.ConfigFile != "" {
		if create {
			// Try to create config file with default values if it doesn't exist
			if cr, err := rt.Ios.AskToCreate(rt.ConfigFile); err != nil {
				// Only return error if error is not one that the file
				// already exists.
				if !errors.Is(err, FileExistsError) {
					// Error occurred during prompt
					return fmt.Errorf("error occurred asking to create config file: %w", err)
				}
			} else if cr {
				// User answered yes
				if err := rt.CreateIfNotExists(rt.ConfigFile); err != nil {
					return fmt.Errorf("failed to create %s: %w", rt.ConfigFile, err)
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
	if rt.ConfigFile != "" {
		err = rt.loadConfigFromFile(rt.ConfigFile)
	} else {
		err = rt.loadMergedConfig()
	}
	if err != nil {
		return err
	}

	return nil
}

// InitLogging initializes logging from runtime configuration.
func (rt *Runtime) InitLogging(cmd *cobra.Command) error {
	// 1. Apply command-line overrides first (highest precedence)
	if cmd.Flags().Changed("log-format") {
		lf, err := cmd.Flags().GetString("log-format")
		if err != nil {
			return fmt.Errorf("failed to fetch flag log-format: %w", err)
		}
		rt.Config.Log.Format = lf
	}
	if cmd.Flags().Changed("log-level") {
		ll, err := cmd.Flags().GetString("log-level")
		if err != nil {
			return fmt.Errorf("failed to fetch flag log-level: %w", err)
		}
		rt.Config.Log.Level = ll
	}
	if cmd.Flags().Changed("log-color") {
		lc, err := cmd.Flags().GetString("log-color")
		if err != nil {
			return fmt.Errorf("failed to fetch flag log-color: %w", err)
		}
		rt.Config.Log.Color = lc
	}

	// 2. Apply defaults for empty values (lowest precedence)
	defaults := config.DefaultGlobalMap()
	if rt.Config.Log.Level == "" {
		rt.Config.Log.Level = defaults["log.level"].(string)
	}
	if rt.Config.Log.Format == "" {
		rt.Config.Log.Format = defaults["log.format"].(string)
	}
	if rt.Config.Log.Color == "" {
		rt.Config.Log.Color = defaults["log.color"].(string)
	}

	// 3. Initialize logger
	if err := log.Init(rt.Config.Log.Level, rt.Config.Log.Format, rt.Config.Log.Color); err != nil {
		return err
	}

	log.Logger.Debug().Msg("logging has been initialized")
	return nil
}

// InitConfigAndLogging initializes both config and logging.
func (rt *Runtime) InitConfigAndLogging(cmd *cobra.Command, createCfg bool) error {
	if err := rt.InitConfig(cmd, createCfg); err != nil {
		return Errorf(CodeConfig, "failed to initialize config: %w", err)
	}
	if err := rt.InitLogging(cmd); err != nil {
		return Errorf(CodeConfig, "failed to initialize logging: %w", err)
	}
	return nil
}

// loadMergedConfig loads system + user config.
func (rt *Runtime) loadMergedConfig() error {
	userPath, err := config.UserConfigPath()
	if err != nil {
		return err
	}
	rt.UserConfigFile = userPath

	cfg, err := config.Load([]config.Source{
		{Name: "system", Path: config.SystemConfigFile, Optional: true},
		{Name: "user", Path: userPath, Optional: true},
	}, config.WithLogger(earlyLogger{logger: log.NewBasicLogger(rt.Ios.Err(), rt.EarlyVerbose, "ochami")}))
	if err != nil {
		return err
	}
	rt.Config = cfg

	ko, err := configfile.EffectiveKoanf(config.SystemConfigFile, userPath)
	if err != nil {
		return err
	}
	rt.Koanf = ko
	return nil
}

// loadConfigFromFile loads a specific config file.
func (rt *Runtime) loadConfigFromFile(path string) error {
	cfg, err := config.LoadFile(path, config.WithLogger(earlyLogger{logger: log.NewBasicLogger(rt.Ios.Err(), rt.EarlyVerbose, "ochami")}))
	if err != nil {
		return err
	}
	rt.Config = cfg

	ko, err := configfile.ReadConfigWithDefaults(path)
	if err != nil {
		return err
	}
	rt.Koanf = ko
	return nil
}

// CreateIfNotExists creates path (a file with optional leading directories) if
// any of the path components do not exist, returning an error if one occurred
// with the creation.
func (rt *Runtime) CreateIfNotExists(path string) error {
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
