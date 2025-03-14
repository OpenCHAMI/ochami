// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/pcs"
	"github.com/elliotchance/pie/v2"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"net/http"
	"strings"
)

// For now use this to map API name to names that make more sense for the CLI, in
// the end we might just move these aliases to the service. Note: We don't report
// status for DistLocking (as the only implementation uses ETCD, so the status
// is just duplicated) or the TaskRunner (as we only use the local implementation)
type commandOutput struct {
	Status       string `json:"pcs,omitempty"`
	KvStore      string `json:"storage,omitempty"`
	StateManager string `json:"smd,omitempty"`
	Vault        string `json:"vault,omitempty"`
}

// format commandOutput as JSON or YAML
func formatOutput(output commandOutput, format string) ([]byte, error) {
	switch strings.ToLower(format) {
	case "json":
		if bytes, err := json.Marshal(output); err != nil {
			return nil, fmt.Errorf("failed to marshal output into JSON: %w", err)
		} else {
			return bytes, nil
		}
	case "yaml":
		if bytes, err := yaml.Marshal(output); err != nil {
			return nil, fmt.Errorf("failed to marshal output into YAML: %w", err)
		} else {
			return bytes, nil
		}
	default:
		return nil, fmt.Errorf("unknown output format: %s", format)
	}
}

// Get the status of PCS either "live" or "ready"
func getStatus(pcsClient *pcs.PCSClient) (string, error) {
	httpEnv, err := pcsClient.GetReadiness()
	if err != nil {
		if errors.Is(err, client.UnsuccessfulHTTPError) {
			log.Logger.Fatal().Err(err).Msg("PCS status (readiness) request yielded unsuccessful HTTP response")
		} else {
			log.Logger.Fatal().Err(err).Msg("failed to get PCS status (readiness)")
		}
	}

	// We are in the "ready" state
	if httpEnv.StatusCode == http.StatusNoContent {
		return "ready", nil
	}

	// If we are not "ready" then check our "liveness"
	httpEnv, err = pcsClient.GetLiveness()
	if err != nil {
		if errors.Is(err, client.UnsuccessfulHTTPError) {
			log.Logger.Fatal().Err(err).Msg("PCS status (liveness) request yielded unsuccessful HTTP response")
		} else {
			log.Logger.Fatal().Err(err).Msg("failed to get PCS status (liveness)")
		}
	}

	// We are in the "live" status
	if httpEnv.StatusCode == http.StatusNoContent {
		return "live", nil
	} else {
		return "", errors.New("unable to get PCS state")
	}
}

// struct used to unmarshall /health endpoint response
type healthOutput struct {
	KvStore      string
	DistLocking  string
	StateManager string
	Vault        string
	TaskRunner   string
}

// allowed flag for status command
func flags() []string {
	return []string{"all", "storage", "smd", "vault"}
}

// pcsStatusCmd represents the pcs-status command
var pcsStatusCmd = &cobra.Command{
	Use:   "status",
	Args:  cobra.NoArgs,
	Short: "Get status of PCS service",
	Run: func(cmd *cobra.Command, args []string) {
		// First and foremost, make sure config is loaded and logging
		// works.
		initConfigAndLogging(cmd, true)

		// Without a base URI, we cannot do anything
		pcsBaseURI, err := getBaseURIPCS(cmd)
		if err != nil {
			log.Logger.Fatal().Err(err).Msg("failed to get base URI for PCS")
		}

		// Create client to make request to PCS
		pcsClient, err := pcs.NewClient(pcsBaseURI, insecure)
		if err != nil {
			log.Logger.Fatal().Err(err).Msg("error creating new PCS client")
		}

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(pcsClient.OchamiClient)

		// Figure out if we need to hit the /health endpoint (only if a flag has been provided)
		flagsProvided := false
		flags := flags()
		for i := 0; i < len(flags); i++ {
			flagsProvided = flagsProvided || cmd.Flag(flags[i]).Changed
		}

		var health healthOutput
		if flagsProvided {
			healthHttpEnv, err := pcsClient.GetHealth()
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Fatal().Err(err).Msg("PCS status (health) request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Fatal().Err(err).Msg("failed to get PCS status (health)")
				}
			}

			// Unmarshall the health
			err = json.Unmarshal(healthHttpEnv.Body, &health)
			if err != nil {
				log.Logger.Fatal().Msg("failed to unmarshal health")
			}
		}

		var output commandOutput
		reportPCSState := !flagsProvided

		// Process the flags and copy the parts we need from the /health
		// endpoint response
		if cmd.Flag("all").Changed {
			output = commandOutput{
				KvStore:      health.KvStore,
				StateManager: health.StateManager,
				Vault:        health.Vault,
			}
			reportPCSState = true
		}
		if cmd.Flag("storage").Changed {
			output.KvStore = health.KvStore
		}
		if cmd.Flag("smd").Changed {
			output.StateManager = health.StateManager
		}
		if cmd.Flag("vault").Changed {
			output.Vault = health.Vault
		}

		// Now deal with the PCS status
		if reportPCSState {
			pcsStatus, err := getStatus(pcsClient)
			if err != nil {
				log.Logger.Fatal().Err(err).Msg("failed to get PCS status")
			}

			output.Status = pcsStatus
		}

		// Print output
		outFmt, err := cmd.Flags().GetString("output-format")
		if err != nil {
			log.Logger.Fatal().Err(err).Msg("failed to get value for --output-format")
		}
		if outBytes, err := formatOutput(output, outFmt); err != nil {
			log.Logger.Fatal().Err(err).Msg("failed to format output")
		} else {
			fmt.Println(string(outBytes))
		}
	},
}

func init() {
	pcsStatusCmd.Flags().Bool("all", false, "print all status data from PCS")
	pcsStatusCmd.Flags().Bool("storage", false, "print status of storage backend from PCS")
	pcsStatusCmd.Flags().Bool("smd", false, "print status of PCS connection to SMD")
	pcsStatusCmd.Flags().Bool("vault", false, "print status of PCS connection to Vault")

	// Mark "all" as mutally exusive of all the other flags
	// First we need a list of flags without "all"
	flags := pie.FilterNot(flags(), func(flag string) bool {
		return flag == "all"
	})
	for i := 0; i < len(flags); i++ {
		pcsStatusCmd.MarkFlagsMutuallyExclusive("all", flags[i])
	}

	pcsStatusCmd.Flags().StringP("output-format", "F", defaultOutputFormat, "format of output printed to standard output")
	pcsCmd.AddCommand(pcsStatusCmd)
}
