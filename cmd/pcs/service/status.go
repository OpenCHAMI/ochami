// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/elliotchance/pie/v2"
	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/pcs"
	"github.com/OpenCHAMI/ochami/pkg/format"

	pcs_lib "github.com/OpenCHAMI/ochami/internal/cli/pcs"
)

const (
	pcsReadyStatus = "ready"
	pcsLiveStatus  = "live"
)

// For now use this to map API name to names that make more sense for the CLI, in
// the end we might just move these aliases to the service. Note: We don't report
// status for DistLocking (as the only implementation uses ETCD, so the status
// is just duplicated) or the TaskRunner (as we only use the local implementation)
type commandOutput struct {
	Status       string `json:"pcs,omitempty" yaml:"pcs,omitempty"`
	KvStore      string `json:"storage,omitempty" yaml:"storage,omitempty"`
	StateManager string `json:"smd,omitempty" yaml:"smd,omitempty"`
	Vault        string `json:"vault,omitempty" yaml:"vault,omitempty"`
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
		return pcsReadyStatus, nil
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
		return pcsLiveStatus, nil
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

func newCmdServiceStatus() *cobra.Command {
	// serviceStatusCmd represents the "pcs service status" command
	var serviceStatusCmd = &cobra.Command{
		Use:   "status",
		Args:  cobra.NoArgs,
		Short: "Get status of Power Control Service (PCS)",
		Long: `Get status of Power Control Service (PCS).

See ochami-pcs(1) for more details.`,
		Example: `  # Get status of PCS
  ochami pcs service status`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			pcsClient := pcs_lib.GetClient(cmd)

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
						log.Logger.Error().Err(err).Msg("PCS status (health) request yielded unsuccessful HTTP response")
					} else {
						log.Logger.Error().Err(err).Msg("failed to get PCS status (health)")
					}
					cli.LogHelpError(cmd)
					os.Exit(1)
				}

				// Unmarshall the health
				err = json.Unmarshal(healthHttpEnv.Body, &health)
				if err != nil {
					log.Logger.Error().Msg("failed to unmarshal health")
					cli.LogHelpError(cmd)
					os.Exit(1)
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
				status, err := getStatus(pcsClient)
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to get PCS status")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}

				output.Status = status
			}

			// Print output
			if outBytes, err := format.MarshalData(output, cli.FormatOutput); err != nil {
				log.Logger.Error().Err(err).Msg("failed to format output")
				cli.LogHelpError(cmd)
				os.Exit(1)
			} else {
				fmt.Println(string(outBytes))
			}
		},
	}

	// Create flags
	serviceStatusCmd.Flags().Bool("all", false, "print all status data from PCS")
	serviceStatusCmd.Flags().Bool("storage", false, "print status of storage backend from PCS")
	serviceStatusCmd.Flags().Bool("smd", false, "print status of PCS connection to SMD")
	serviceStatusCmd.Flags().Bool("vault", false, "print status of PCS connection to Vault")

	// Mark "all" as mutally exusive of all the other flags
	// First we need a list of flags without "all"
	flags := pie.FilterNot(flags(), func(flag string) bool {
		return flag == "all"
	})
	for i := 0; i < len(flags); i++ {
		serviceStatusCmd.MarkFlagsMutuallyExclusive("all", flags[i])
	}

	serviceStatusCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")
	serviceStatusCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return serviceStatusCmd
}
