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
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/config"
	"github.com/openchami/ochami/pkg/format"
)

// Runtime holds the invocation-owned state for the CLI. It replaces the
// previous global variables (Ios, FormatInput, FormatOutput, Token, ConfigFile,
// activeConfig, activeKoanf) to enable test isolation and parallel test execution.
//
// A *Runtime is owned by a single command invocation (one per NewRuntime or
// NewTestRuntime call) and is not safe to share across goroutines or parallel
// subtests without external synchronization.
type Runtime struct {
	// I/O streams
	Ios *IOStreams
	// Env provides invocation-owned environment lookup.
	Env Environment
	// FileCreation provides the filesystem operations used when creating a
	// configuration file.
	FileCreation FileCreationOperations
	// Logger writes diagnostics for this invocation only.
	Logger zerolog.Logger

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

// Environment is the process-environment surface used by Runtime.
type Environment interface {
	LookupEnv(string) (string, bool)
}

// EnvironmentFunc adapts a function such as os.LookupEnv to Environment.
type EnvironmentFunc func(string) (string, bool)

// LookupEnv implements Environment.
func (f EnvironmentFunc) LookupEnv(key string) (string, bool) { return f(key) }

// FileCreationOperations is the narrow filesystem surface used to create a
// configuration file and its parent directories.
type FileCreationOperations interface {
	Stat(string) error
	MkdirAll(string, os.FileMode) error
	OpenFile(string, int, os.FileMode) (io.Closer, error)
}

type osFileCreationOperations struct{}

func (osFileCreationOperations) Stat(path string) error {
	_, err := os.Stat(path)
	return err
}

func (osFileCreationOperations) MkdirAll(path string, mode os.FileMode) error {
	return os.MkdirAll(path, mode)
}

func (osFileCreationOperations) OpenFile(path string, flag int, mode os.FileMode) (io.Closer, error) {
	return os.OpenFile(path, flag, mode)
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

// ConfirmCreate asks whether a missing file should be created. Filesystem
// existence checks are owned by Runtime rather than the I/O streams.
func (i *IOStreams) ConfirmCreate(path string) (bool, error) {
	create, err := i.LoopYesNo(fmt.Sprintf("%s does not exist. Create it?", path))
	if err != nil {
		return false, fmt.Errorf("error fetching user input: %w", err)
	}
	return create, nil
}

// LoopYesNo takes prompt p and appends " [yn]:" to it and prompts the user for
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

// noEnv is an Environment that reports every variable as unset. It is the
// default for NewTestRuntime so tests are hermetic (isolated from the host's
// real environment and safe under t.Parallel()) unless a test explicitly
// opts in via WithEnvironment.
var noEnv = EnvironmentFunc(func(string) (string, bool) { return "", false })

// newBaseRuntime constructs the fields shared by NewRuntime and NewTestRuntime,
// so the two constructors can't silently diverge as fields are added.
//
// Logger defaults to a plain writer to the runtime's own stderr stream rather
// than a no-op logger. The runtime is attached to the command context (and so
// reachable via LoggerFromCommand/FromContext) before InitConfigAndLogging
// runs, and again if it fails before InitLogging replaces this default with a
// fully configured logger — a no-op default would silently swallow exactly
// that failure's error report.
func newBaseRuntime(ios *IOStreams) *Runtime {
	return &Runtime{
		Ios:          ios,
		FileCreation: osFileCreationOperations{},
		Logger:       zerolog.New(ios.Err()).Level(zerolog.WarnLevel).With().Timestamp().Logger(),
		FormatInput:  format.DataFormatJson,
		FormatOutput: format.DataFormatJson,
	}
}

// NewRuntime creates a new Runtime instance for production use.
// It owns the process streams, environment lookup, and default formats for one
// invocation. Tests should use NewTestRuntime and inject the runtime into the
// command context.
func NewRuntime() *Runtime {
	rt := newBaseRuntime(NewIOStreams(os.Stdin, os.Stdout, os.Stderr))
	rt.Env = EnvironmentFunc(os.LookupEnv)
	rt.LoadConfig = true
	return rt
}

// NewTestRuntime creates a new Runtime instance for testing.
// It accepts custom I/O streams and allows setting formats, token, and config
// explicitly. Its environment is hermetic by default (see noEnv); use
// WithEnvironment to inject specific variables.
func NewTestRuntime(stdin io.Reader, stdout, stderr io.Writer) *Runtime {
	rt := newBaseRuntime(NewIOStreams(stdin, stdout, stderr))
	rt.Env = noEnv
	// Set a reasonable timeout for tests (30 seconds) to match the default
	// config. This prevents context deadline errors when clients use
	// rt.Config.Timeout.
	rt.Config = config.Config{Timeout: 30 * time.Second}
	return rt
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

// WithEnvironment sets the environment used by the runtime.
func (rt *Runtime) WithEnvironment(env Environment) *Runtime {
	rt.Env = env
	return rt
}

// WithLogger sets the logger used by this runtime.
func (rt *Runtime) WithLogger(logger zerolog.Logger) *Runtime {
	rt.Logger = logger
	return rt
}

func (rt *Runtime) lookupEnv(key string) (string, bool) {
	if rt.Env == nil {
		return os.LookupEnv(key)
	}
	return rt.Env.LookupEnv(key)
}

// ResolveUserConfigFile records the platform-specific user configuration path
// without reading that file.
func (rt *Runtime) ResolveUserConfigFile() error {
	path, err := config.UserConfigPathWithEnv(rt.lookupEnv)
	if err != nil {
		return err
	}
	rt.UserConfigFile = path
	return nil
}

// runtimeKey is the context key for storing Runtime instances.
type runtimeKey struct{}

// ContextWithRuntime returns a copy of ctx carrying rt, retrievable via
// FromContext or RuntimeFromCommand. It is a package function rather than a
// *Runtime method (unlike Runtime's other With* methods, which all return
// *Runtime) so its return type isn't misleading by analogy to those or to
// http.Request.WithContext.
func ContextWithRuntime(ctx context.Context, rt *Runtime) context.Context {
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
//
// This is a pure retrieval: it does not mutate the runtime. Format flags are
// applied exactly once, by the root command's PersistentPreRunE (which runs
// before any leaf command's RunE), so callers do not need to call
// ApplyFormatFlags themselves.
func RuntimeFromCommand(cmd *cobra.Command) (*Runtime, error) {
	if cmd == nil {
		return nil, Errorf(CodeConfig, "CLI runtime is unavailable: command is nil")
	}
	rt, ok := FromContext(cmd.Context())
	if !ok {
		return nil, Errorf(CodeConfig, "CLI runtime is unavailable; execute commands through the root command")
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
			return Errorf(CodeAuth, "token is expired: %w", err)
		} else if errors.Is(err, jwt.TokenNotYetValidError()) {
			return Errorf(CodeAuth, "token is not yet valid (nbf in future): %w", err)
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
		rt.Logger.Warn().Msgf("%s until token expires", exp.Sub(now))
	}

	return nil
}

// SetTokenFromFlag sets the access token from the --token flag.
func (rt *Runtime) SetTokenFromFlag(cmd *cobra.Command) error {
	if f := cmd.Flag("token"); f != nil && f.Changed {
		rt.Token = f.Value.String()
		rt.Logger.Debug().Msg("--token passed, setting token to its value: " + client.RedactToken(rt.Token, rt.ShowToken(cmd)))
		return nil
	}
	return nil
}

// SetTokenFromEnv sets the access token from environment variable based on cluster name.
func (rt *Runtime) SetTokenFromEnv(cmd *cobra.Command) error {
	var clusterName string
	if f := cmd.Flag("cluster"); f != nil && f.Changed {
		clusterName = f.Value.String()
		rt.Logger.Debug().Msg("--cluster specified: " + clusterName)
	} else if rt.Config.DefaultCluster != "" {
		clusterName = rt.Config.DefaultCluster
		rt.Logger.Debug().Msg("--cluster not specified, using default-cluster: " + clusterName)
	} else {
		return Errorf(CodeAuth, "no default-cluster specified and --token not passed")
	}

	varPrefix := strings.ReplaceAll(clusterName, "-", "_")
	varPrefix = strings.ReplaceAll(varPrefix, " ", "_")

	envVarToRead := strings.ToUpper(varPrefix) + "_ACCESS_TOKEN"
	rt.Logger.Debug().Msg("Reading token from environment variable: " + envVarToRead)
	if t, tokenSet := rt.lookupEnv(envVarToRead); tokenSet {
		rt.Logger.Debug().Msgf("Token found from environment variable: %s=%s", envVarToRead, client.RedactToken(t, rt.ShowToken(cmd)))
		rt.Token = t
		return nil
	}

	return Errorf(CodeAuth, "environment variable %s unset for reading token for cluster %q", envVarToRead, clusterName)
}

// HandleToken handles token reading and validation using runtime state.
func (rt *Runtime) HandleToken(cmd *cobra.Command) error {
	if f := cmd.Flag("no-token"); f != nil && f.Value.String() == "true" {
		// --no-token overrides any cluster settings
		rt.Logger.Debug().Msg("--no-token passed, not reading or checking for token")
		return nil
	}

	// Check if enable-auth is set for cluster and only read/check
	// token if true
	var clusterName string
	if f := cmd.Flag("cluster"); f != nil && f.Changed {
		// Use cluster passed via --cluster
		clusterName = f.Value.String()
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
				rt.Logger.Debug().Msgf("authentication enabled for cluster %s, reading and checking token", cl.Name)
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
				rt.Logger.Debug().Msgf("authentication disabled for cluster %s, not reading or checking for token", cl.Name)
			}
		}
	}
	return nil
}

// UseCACert configures client with CA certificate from runtime.
func (rt *Runtime) UseCACert(ochamiClient *client.OchamiClient) error {
	if rt.CACertPath != "" {
		rt.Logger.Debug().Msgf("Attempting to use CA certificate at %s", rt.CACertPath)
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
			rt.Logger.Warn().Err(err).Msgf("failed to get timeout from flag, falling back to config value of %s", rt.Config.Timeout)
		}
	}
	return rt.Config.Timeout
}

// HandlePayload unmarshals raw data or data from a payload file into v for
// command cmd if --data and, optionally, --format-input, are passed.
func (rt *Runtime) HandlePayload(cmd *cobra.Command, v any) error {
	if cmd.Flag("data").Changed {
		data := cmd.Flag("data").Value.String()
		if err := client.ReadPayloadWithReader(data, rt.Ios.In(), rt.FormatInput, v); err != nil {
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
		if err := client.ReadPayloadSliceWithReader[T](data, rt.Ios.In(), rt.FormatInput, v); err != nil {
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

// resolveCluster determines which configured cluster (if any) applies to cmd,
// following the same --cluster / default-cluster precedence used by
// GetBaseURI and GetAPIVersion. found reports whether a cluster name was
// selected (via --cluster or default-cluster) AND located in rt.Config.Clusters;
// it is tracked explicitly via the loop match rather than by comparing the
// result against a zero-valued config.ConfigCluster{}, so this code keeps
// working correctly even if config.ConfigClusterConfig later gains a field
// that isn't comparable with ==.
func (rt *Runtime) resolveCluster(cmd *cobra.Command) (name string, cc config.ConfigClusterConfig, found bool, err error) {
	var clusterName string
	if cmd.Flag("cluster").Changed {
		// An explicit cluster overrides the configured default cluster.
		clusterName = cmd.Flag("cluster").Value.String()
	} else if rt.Config.DefaultCluster != "" {
		clusterName = rt.Config.DefaultCluster
	} else {
		return "", config.ConfigClusterConfig{}, false, nil
	}

	for _, c := range rt.Config.Clusters {
		if c.Name == clusterName {
			return clusterName, c.Cluster, true, nil
		}
	}
	return clusterName, config.ConfigClusterConfig{}, false, fmt.Errorf("cluster %s not found", clusterName)
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
	clusterName, clusterConfig, found, err := rt.resolveCluster(cmd)
	if err != nil {
		return "", err
	}
	if found {
		if cmd.Flag("cluster").Changed {
			rt.Logger.Debug().Msgf("reading URI from cluster %s passed from command line", clusterName)
		} else {
			rt.Logger.Debug().Msgf("using base URI from default cluster %s", clusterName)
		}
	}
	// Check flags (--cluster-uri and/or --uri) and override any
	// previously-set values while leaving unspecified ones alone.
	if cmd.Flag("cluster-uri").Changed || (cmd.Flag("uri") != nil && cmd.Flag("uri").Changed) {
		rt.Logger.Debug().Msg("using base URI passed on command line")
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
	var apiVersion string
	clusterName, clusterConfig, found, err := rt.resolveCluster(cmd)
	if err != nil {
		return "", err
	}
	if found {
		if cmd.Flag("cluster").Changed {
			rt.Logger.Debug().Msgf("reading API version for %s from cluster %s passed from command line", serviceName, clusterName)
		} else {
			rt.Logger.Debug().Msgf("using API version from %s in default cluster %s", serviceName, clusterName)
		}
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
	// Do not read or write config file if --ignore-config passed
	if f := cmd.Flag("ignore-config"); f != nil && f.Value.String() == "true" {
		ko, cfg, err := config.LoadDefaultsKoanf(rt.configLoadOpts()...)
		if err != nil {
			return fmt.Errorf("unable to load default config: %w", err)
		}
		rt.Config = cfg
		rt.Koanf = ko
		// Resolve (but do not read) the user config path even under
		// --ignore-config, so commands that report or target it (e.g. "config
		// show --user") still have a usable path instead of an empty string.
		if rt.UserConfigFile == "" {
			if err := rt.ResolveUserConfigFile(); err != nil {
				return fmt.Errorf("unable to resolve user config path: %w", err)
			}
		}
		return nil
	}

	if rt.ConfigFile != "" {
		if create {
			// Try to create config file with default values if it doesn't exist
			if cr, err := rt.AskToCreate(rt.ConfigFile); err != nil {
				// Only return error if error is not one that the file
				// already exists.
				if !errors.Is(err, ErrFileExists) {
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
	logger, err := log.New(rt.Ios.Err(), rt.Config.Log.Level, rt.Config.Log.Format, rt.Config.Log.Color)
	if err != nil {
		return err
	}
	rt.Logger = logger

	rt.Logger.Debug().Msg("logging has been initialized")
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

// loadMergedConfig loads system + user config. rt.Config and rt.Koanf are
// derived from a single underlying load (via config.LoadKoanf) so the two can
// never observe different file contents, and --verbose tracing covers the
// load that actually produced them. The source list is built here (rather
// than via config.LoadMergedKoanf) only because the user config path must be
// resolved through rt.lookupEnv for hermetic-environment testability.
func (rt *Runtime) loadMergedConfig() error {
	userPath, err := config.UserConfigPathWithEnv(rt.lookupEnv)
	if err != nil {
		return err
	}
	rt.UserConfigFile = userPath

	ko, cfg, err := config.LoadKoanf([]config.Source{
		{Name: "default", Map: config.DefaultGlobalMap()},
		{Name: "system", Path: config.SystemConfigFile, Optional: true},
		{Name: "user", Path: userPath, Optional: true},
	}, rt.configLoadOpts()...)
	if err != nil {
		return err
	}
	rt.Config = cfg
	rt.Koanf = ko
	return nil
}

// loadConfigFromFile loads a specific config file. rt.Config and rt.Koanf are
// derived from a single underlying load (via config.LoadFileKoanf) so the two
// can never observe different file contents.
func (rt *Runtime) loadConfigFromFile(path string) error {
	ko, cfg, err := config.LoadFileKoanf(path, rt.configLoadOpts()...)
	if err != nil {
		return err
	}
	rt.Config = cfg
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
	if err := rt.FileCreation.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("checking %s failed: %w", path, err)
	}

	parentDir := filepath.Dir(path)
	if err := rt.FileCreation.MkdirAll(parentDir, 0o755); err != nil {
		return fmt.Errorf("could not create parent dir %s: %w", parentDir, err)
	}
	f, err := rt.FileCreation.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("creating %s failed: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing %s failed: %w", path, err)
	}

	return nil
}

// AskToCreate checks whether path is missing and, if so, asks the user whether
// it should be created. Existing files are reported with ErrFileExists;
// other stat failures are preserved.
func (rt *Runtime) AskToCreate(path string) (bool, error) {
	if path == "" {
		return false, fmt.Errorf("path cannot be empty")
	}
	if err := rt.FileCreation.Stat(path); err == nil {
		return false, ErrFileExists
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("checking %s failed: %w", path, err)
	}
	return rt.Ios.ConfirmCreate(path)
}
