// SPDX-FileCopyrightText: © 2024-2026 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package status

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	pcs_lib "github.com/openchami/ochami/internal/cli/pcs"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/format"
)

var xnames []string

type PowerFilter string

var powerFilter PowerFilter = ""

const (
	powerOn        PowerFilter = "on"
	powerOff       PowerFilter = "off"
	powerUndefined PowerFilter = "undefined"
)

func (l *PowerFilter) String() string {
	return string(*l)
}

func (l *PowerFilter) Set(value string) error {
	switch strings.ToLower(value) {
	case "on", "off", "undefined":
		*l = PowerFilter(strings.ToLower(value))
		return nil
	default:
		return fmt.Errorf("invalid power filter: %s (must be on, off, or undefined)", value)
	}
}

func (l PowerFilter) Type() string {
	return "PowerFilter"
}

var (
	powerFilterHelp = map[string]string{
		string(powerOn):        "Include components that are powered on",
		string(powerOff):       "Include components that are powered off",
		string(powerUndefined): "Include components with undefined power state",
	}
)

func pcsStatusListPowerFilterCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var helpSlice []string
	for k, v := range powerFilterHelp {
		helpSlice = append(helpSlice, fmt.Sprintf("%s\t%s", k, v))
	}
	return helpSlice, cobra.ShellCompDirectiveNoFileComp
}

type MgmtFilter string

var mgmtFilter MgmtFilter = ""

func (l *MgmtFilter) String() string {
	return string(*l)
}

func (l *MgmtFilter) Set(value string) error {
	switch strings.ToLower(value) {
	case "available", "unavailable":
		*l = MgmtFilter(strings.ToLower(value))
		return nil
	default:
		return fmt.Errorf("invalid management filter: %s (must be available or unavailable)", value)
	}
}

func (l MgmtFilter) Type() string {
	return "MgmtFilter"
}

var (
	mgmtFilterHelp = map[string]string{
		"available":   "Include components that are available",
		"unavailable": "Include components that are unavailable",
	}
)

func pcsStatusListMgmtFilterCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var helpSlice []string
	for k, v := range mgmtFilterHelp {
		helpSlice = append(helpSlice, fmt.Sprintf("%s\t%s", k, v))
	}
	return helpSlice, cobra.ShellCompDirectiveNoFileComp
}

func newCmdStatusList() *cobra.Command {

	// pcsStatusListCmd represents the "pcs status list" command
	var pcsStatusListCmd = &cobra.Command{
		Use:   "list",
		Args:  cobra.NoArgs,
		Short: "List active PCS transitions",
		Long: `List active PCS transitions.

See ochami-pcs(1) for more details.`,
		Example: `  # List status
  ochami pcs status list`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			pcsClient := pcs_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Get status
			statusHttpEnv, err := pcsClient.GetStatus(xnames, string(powerFilter), string(mgmtFilter), cli.Token)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("PCS status request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to list PCS transitions")
				}
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			var output interface{}
			err = json.Unmarshal(statusHttpEnv.Body, &output)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to unmarshal status response")
				cli.LogHelpError(cmd)
				os.Exit(1)
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

	// Define flags
	pcsStatusListCmd.Flags().StringSliceVarP(&xnames, "xname", "x", []string{}, "one or more xnames to get the status for")
	pcsStatusListCmd.Flags().VarP(&powerFilter, "power-filter", "p", "filter results by power state (on, off, undefined)")
	pcsStatusListCmd.RegisterFlagCompletionFunc("power-filter", pcsStatusListPowerFilterCompletion)
	pcsStatusListCmd.Flags().VarP(&mgmtFilter, "mgmt-filter", "m", "filter results by management state (available, unavailable)")
	pcsStatusListCmd.RegisterFlagCompletionFunc("mgmt-filter", pcsStatusListMgmtFilterCompletion)

	pcsStatusListCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")
	pcsStatusListCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return pcsStatusListCmd
}
