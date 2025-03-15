// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

// lib.go provides library functions to the cmd package, a.k.a. all cobra
// commands.

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
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/spf13/cobra"
)

// Set log level verbosity based on config file (log.level) or --log-level.
// The command line option overrides the config file option.
func initLogging() {
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

// askToCreate prompts the user to, if path does not exist, to create a blank
// file at path. If it exists, nil is returned. If the user declines, a
// UserDeclinedError is returned. If an error occurs during creation, an error
// is returned.
func askToCreate(path string) error {
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

func initConfig() {
	// Do not read or write config file if --ignore-config passed
	if rootCmd.Flag("ignore-config").Changed {
		return
	}

	if configFile != "" {
		// Try to create config file with default values if it doesn't exist
		if err := askToCreate(configFile); err != nil {
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

func getBaseURI(cmd *cobra.Command) (string, error) {
       // Precedence of getting base URI for requests:
       //
       // 1. If --cluster is set, search config file for matching name and read
       //    details from there.
       // 2. If flags corresponding to cluster info (e.g. --base-uri) are set,
       //    read details from them.
       // 3. If "default-cluster" is set in config file (config file must be
       //    specified), use cluster identified by that name as source of info.
       // 4. Data sources exhausted, err.
       var (
               clusterList  []config.ConfigCluster
               clusterToUse config.ConfigCluster
               clusterName  string
       )
       if cmd.Flag("cluster").Changed {
               clusterList = config.GlobalConfig.Clusters
               clusterName = cmd.Flag("cluster").Value.String()
               log.Logger.Debug().Msgf("using base URI from cluster %s passed from command line", clusterName)
               for _, c := range clusterList {
                       if c.Name == clusterName {
                               clusterToUse = c
                               break
                       }
               }
               if clusterToUse == (config.ConfigCluster{}) {
                       return "", fmt.Errorf("cluster %s not found", clusterName)
               }
               if clusterToUse.Cluster.BaseURI == "" {
                       return "", fmt.Errorf("base-uri not set for cluster %s specified with --cluster", clusterName)
               }

               log.Logger.Debug().Msgf("base URI: %s", clusterToUse.Cluster.BaseURI)

               return clusterToUse.Cluster.BaseURI, nil
       } else if cmd.Flag("base-uri").Changed {
               log.Logger.Debug().Msg("using base URI passed on command line")
               log.Logger.Debug().Msgf("base URI: %s", baseURI)
               return baseURI, nil
       } else if config.GlobalConfig.DefaultCluster != "" {
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

               log.Logger.Debug().Msgf("base URI: %s", clusterToUse.Cluster.BaseURI)

               return clusterToUse.Cluster.BaseURI, nil
       }

       return "", fmt.Errorf("no base-uri set bia --base-uri, --cluster, or config file")
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

// printUsageHandleError is a simple wrapper around printing a command's usage
// that handles errors.
func printUsageHandleError(cmd *cobra.Command) {
	if err := cmd.Usage(); err != nil {
		log.Logger.Error().Err(err).Msg("failed to print usage")
		os.Exit(1)
	}
}
