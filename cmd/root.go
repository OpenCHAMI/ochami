// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OpenCHAMI/ochami/internal/config"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/internal/version"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/spf13/cobra"
)

const (
	defaultPayloadFormat = "json"
	defaultOutputFormat  = "json"
)

var (
	// Errors
	UserDeclinedError = fmt.Errorf("user declined")

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
	Use:     config.ProgName,
	Args:    cobra.NoArgs,
	Short:   "Command line interface for interacting with OpenCHAMI services",
	Long:    "",
	Version: version.Version,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			err := cmd.Usage()
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to print usage")
				os.Exit(1)
			}
			os.Exit(0)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Logger.Error().Err(err).Msg("failed to execute root command")
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(
		InitConfig,
		InitLogging,
	)
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "path to configuration file to use")
	rootCmd.PersistentFlags().StringP("log-format", "L", "", "log format (json,rfc3339,basic)")
	rootCmd.PersistentFlags().StringP("log-level", "l", "", "set verbosity of logs (info,warning,debug)")
	rootCmd.PersistentFlags().StringP("cluster", "C", "", "name of cluster whose config to use for this command")
	rootCmd.PersistentFlags().StringP("api-uri", "u", "", "base URI for OpenCHAMI services, excluding service base path")
	rootCmd.PersistentFlags().StringVar(&cacertPath, "cacert", "", "path to root CA certificate in PEM format")
	rootCmd.PersistentFlags().StringVarP(&token, "token", "t", "", "access token to present for authentication")
	rootCmd.PersistentFlags().BoolVarP(&insecure, "insecure", "k", false, "do not verify TLS certificates")
	rootCmd.PersistentFlags().Bool("ignore-config", false, "do not use any config file")
	rootCmd.PersistentFlags().BoolVarP(&config.EarlyVerbose, "verbose", "v", false, "be verbose before logging is initialized")

	// Either use cluster from config file or specify details on CLI
	rootCmd.MarkFlagsMutuallyExclusive("cluster", "api-uri")
}

// Set log level verbosity based on config file (log.level) or --log-level.
// The command line option overrides the config file option.
func InitLogging() {
	if rootCmd.PersistentFlags().Lookup("log-format").Changed {
		lf, err := rootCmd.PersistentFlags().GetString("log-format")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to fetch flag log-format: %v\n", config.ProgName, err)
			os.Exit(1)
		}
		config.GlobalConfig.Log.Format = lf
	}
	if rootCmd.PersistentFlags().Lookup("log-level").Changed {
		ll, err := rootCmd.PersistentFlags().GetString("log-level")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: failed to fetch flag log-level: %v\n", config.ProgName, err)
			os.Exit(1)
		}
		config.GlobalConfig.Log.Level = ll
	}

	if err := log.Init(config.GlobalConfig.Log.Level, config.GlobalConfig.Log.Format); err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to initialize logger: %v\n", config.ProgName, err)
		os.Exit(1)
	}

	log.Logger.Debug().Msg("logging has been initialized")
}

// AskToCreate prompts the user to, if path does not exist, to create a blank
// file at path. If it exists, nil is returned. If the user declines, a
// UserDeclinedError is returned. If an error occurs during creation, an error
// is returned.
func AskToCreate(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		respConfigCreate := loopYesNo(fmt.Sprintf("%s does not exist. Create it?", path))
		if respConfigCreate {
			parentDir := filepath.Dir(path)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("could not create parent dir %s: %w", parentDir, err)
			}
			f, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE, 0644)
			if err != nil {
				return fmt.Errorf("creating %s failed: %w", path, err)
			}
			f.Close()
		} else {
			return UserDeclinedError
		}
	}

	return nil
}

func InitConfig() {
	// Do not read or write config file if --ignore-config passed
	if rootCmd.Flag("ignore-config").Changed {
		return
	}

	if configFile != "" {
		// Try to create config file with default values if it doesn't exist
		if err := AskToCreate(configFile); err != nil {
			if errors.Is(err, UserDeclinedError) {
				fmt.Fprintf(os.Stderr, "%s: user declined to create file; exiting...\n", config.ProgName)
				os.Exit(0)
			} else {
				fmt.Fprintf(os.Stderr, "%s: failed to create %s: %v\n", config.ProgName, configFile, err)
				os.Exit(1)
			}
		}
	}

	// Read configuration from file, if passed or merge config from system
	// config file and user config file if not passed.
	err := config.LoadConfig(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: failed to load configuration: %v\n", config.ProgName, err)
		os.Exit(1)
	}
}

// prompt displays a text prompt and returns what the user entered. It continues
// to repeat the prompt as long as the user input is empty.
func prompt(prompt string) string {
	var s string
	resp := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprint(os.Stderr, prompt+" ")
		s, _ = resp.ReadString('\n')
		if s != "" {
			break
		}
	}
	return strings.TrimSpace(s)
}

// loopYesNo takes prompt p and appends " [yN]: " to it and prompts the user for
// input. As long as the user's input is not "y" or "n" (case insensitive), the
// function redisplays the prompt. If the user's response is "y", true is
// returned. If the user's response is "n", false is returned.
func loopYesNo(p string) bool {
	for {
		resp := prompt(fmt.Sprintf("%s [yN]:", p))
		switch strings.ToLower(resp) {
		case "y":
			return true
		case "n":
			return false
		default:
			continue
		}
	}
}

// checkToken takes a pointer to a Cobra command and checks to see if --token
// was set. If not, an error is printed and the program exits.
func checkToken(cmd *cobra.Command) {
	// TODO: Check token validity/expiration
	if token == "" {
		log.Logger.Error().Msg("no token set")
		os.Exit(1)
	}

	// Try to parse token
	t, err := jwt.ParseString(token, jwt.WithValidate(false))
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

// useCACert takes a pointer to a client.OchamiClient and, if a path to a CA
// certificate has been set via --cacert, it configures it to use it. If an
// error occurs, a log is printed and the program exits.
func useCACert(client *client.OchamiClient) {
	if cacertPath != "" {
		log.Logger.Debug().Msgf("Attempting to use CA certificate at %s", cacertPath)
		if err := client.UseCACert(cacertPath); err != nil {
			log.Logger.Error().Err(err).Msgf("failed to load CA certificate %s", cacertPath)
			os.Exit(1)
		}
	}
}

func getBaseURI(cmd *cobra.Command, serviceName config.ServiceName) (string, error) {
	// Precedence of getting base URI for requests (higher numbers override
	// all preceding numbers):
	//
	// 1. If "default-cluster" is set in config file (config file must be
	//    specified), use cluster identified by that name as source of info.
	// 2. If --cluster is set, search config file for matching name and read
	//    details from there.
	// 3. If flags corresponding to cluster info (e.g. --api-uri,
	//    --<service>-uri) are set, read details from them.
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
	// 1. Check flags (--api-uri and/or --uri) and override any
	// previously-set values while leaving unspecified ones alone.
	if cmd.Flag("api-uri").Changed || (cmd.Flag("uri") != nil && cmd.Flag("uri").Changed) {
		log.Logger.Debug().Msg("using base URI passed on command line")
		ccc := config.ConfigClusterConfig{APIURI: cmd.Flag("api-uri").Value.String()}
		switch serviceName {
		case config.ServiceBSS:
			ccc.BSSURI = cmd.Flag("uri").Value.String()
		case config.ServiceCloudInit:
			ccc.CloudInitURI = cmd.Flag("uri").Value.String()
		case config.ServicePCS:
			ccc.PCSURI = cmd.Flag("uri").Value.String()
		case config.ServiceSMD:
			ccc.SMDURI = cmd.Flag("uri").Value.String()
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

// setTokenFromEnvVar sets the access token for a cobra command cmd. If --token
// was passed, that value is set as the access token. Otherwise, the token is
// read from an environment variable whose format is <CLUSTER>_ACCESS_TOKEN
// where <CLUSTER> is the name of the cluster, in upper case, being contacted.
// The value of <CLUSTER> is determined by taking the cluster name, passed
// either by --cluster or reading default-cluster from the config file (the
// former preceding the latter), replacing spaces and dashes (-) with
// underscores, and making the letters uppercase. If no config file is set or
// the environment variable is not set, an error is logged and the program
// exits.
func setTokenFromEnvVar(cmd *cobra.Command) {
	var (
		clusterName string
		varPrefix   string
	)
	if cmd.Flag("token").Changed {
		token = cmd.Flag("token").Value.String()
		log.Logger.Debug().Msg("--token passed, setting token to its value: " + token)
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
		os.Exit(1)
	}

	varPrefix = strings.ReplaceAll(clusterName, "-", "_")
	varPrefix = strings.ReplaceAll(varPrefix, " ", "_")

	envVarToRead := strings.ToUpper(varPrefix) + "_ACCESS_TOKEN"
	log.Logger.Debug().Msg("Reading token from environment variable: " + envVarToRead)
	if t, tokenSet := os.LookupEnv(envVarToRead); tokenSet {
		log.Logger.Debug().Msgf("Token found from environment variable: %s=%s", envVarToRead, t)
		token = t
		return
	}

	log.Logger.Error().Msgf("Environment variable %s unset for reading token for cluster %q", envVarToRead, clusterName)
	os.Exit(1)
}

// handlePayload unmarshals a payload file into data for command cmd if
// --payload and, optionally, --payload-format, are passed.
func handlePayload(cmd *cobra.Command, data any) {
	if cmd.Flag("payload").Changed {
		dFile := cmd.Flag("payload").Value.String()
		dFormat := cmd.Flag("payload-format").Value.String()
		err := client.ReadPayload(dFile, dFormat, data)
		if err != nil {
			log.Logger.Error().Err(err).Msg("unable to read payload for request")
			os.Exit(1)
		}
	}
}
