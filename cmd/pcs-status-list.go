// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/format"
)

var statusXnames []string

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
		pcsClient := pcsGetClient(cmd)

		// Handle token for this command
		handleToken(cmd)

		// Get status
		statusHttpEnv, err := pcsClient.GetStatus(statusXnames, string(powerFilter), string(mgmtFilter), token)
		if err != nil {
			if errors.Is(err, client.UnsuccessfulHTTPError) {
				log.Logger.Error().Err(err).Msg("PCS status request yielded unsuccessful HTTP response")
			} else {
				log.Logger.Error().Err(err).Msg("failed to list PCS transitions")
			}
			logHelpError(cmd)
			os.Exit(1)
		}

		var output interface{}
		err = json.Unmarshal(statusHttpEnv.Body, &output)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to unmarshal status response")
			logHelpError(cmd)
			os.Exit(1)
		}

		// Print output
		if outBytes, err := format.MarshalData(output, formatOutput); err != nil {
			log.Logger.Error().Err(err).Msg("failed to format output")
			logHelpError(cmd)
			os.Exit(1)
		} else {
			fmt.Println(string(outBytes))
		}
	},
}

func init() {
	pcsStatusListCmd.Flags().StringSliceVarP(&statusXnames, "xname", "x", []string{}, "one or more xnames to get the status for")
	pcsStatusListCmd.Flags().VarP(&powerFilter, "power-filter", "p", "filter results by power state (on, off, undefined)")
	pcsStatusListCmd.RegisterFlagCompletionFunc("power-filter", pcsStatusListPowerFilterCompletion)
	pcsStatusListCmd.Flags().VarP(&mgmtFilter, "mgmt-filter", "m", "filter results by management state (available, unavailable)")
	pcsStatusListCmd.RegisterFlagCompletionFunc("mgmt-filter", pcsStatusListMgmtFilterCompletion)

	pcsStatusListCmd.Flags().VarP(&formatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")
	pcsStatusListCmd.RegisterFlagCompletionFunc("format-output", completionFormatData)

	pcsStatusCmd.AddCommand(pcsStatusListCmd)
}
