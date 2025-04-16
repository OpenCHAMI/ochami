// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/smd"
	"github.com/openchami/schemas/schemas/csm"
	"github.com/spf13/cobra"
)

// rfeAddCmd represents the smd-rfe-add command
var rfeAddCmd = &cobra.Command{
	Use:   "add (-d (<payload_data> | @<payload_file>)) | (<xname> <name> <ip_addr> <mac_addr>)",
	Args:  cobra.MaximumNArgs(4),
	Short: "Add new redfish endpoint(s)",
	Long: `Add new redfish endpoint(s). An xname, name, IP address, and MAC
address are required. Alternatively, pass -d to pass raw
payload data or (if flag argument starts with @) a file
containing the payload data. -f can be specified to change
the format of the input payload data ('json' by default),
but the rules above still apply for the payload. If "-" is
used as the input payload filename, the data is read from
standard input.

This command sends a POST to SMD. An access token is required.

See ochami-smd(1) for more details.`,
	Example: `  # Add redfish endpoint using CLI flags
  ochami smd rfe add x3000c1s7b56 bmc-node56 172.16.0.156 de:ca:fc:0f:fe:ee

  # Add redfish endpoints using input payload data
  ochami smd rfe add -d '{
    "RedfishEndpoints": [
      {
        "ID": "x3000c1s7b56",
	"Type": "NodeBMC",
	"Name": "bmc-node56",
	"IPAddress": "172.16.0.156",
	"MACAddr": "de:ca:fc:0f:fe:ee",
	"Enabled": true
      }
    ]
  }'

  # Add redfish endpoints using input payload file
  ochami smd rfe add -d @payload.json
  ochami smd rfe add -d @payload.yaml -f yaml

  # Add redfish endpoints using data from standard input
  echo '<json_data>' | ochami smd rfe add -d @-
  echo '<yaml_data>' | ochami smd rfe add -d @- -f yaml`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Check that all required args are passed
		if len(args) == 0 && !cmd.Flag("data").Changed {
			printUsageHandleError(cmd)
			os.Exit(0)
		} else if len(args) > 4 {
			return fmt.Errorf("expected 4 arguments (xname, name, ip_addr, mac_addr) but got %d: %v", len(args), args)
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Create client to use for requests
		smdClient := smdGetClient(cmd, true)

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(smdClient.OchamiClient)

		var rfes smd.RedfishEndpointSlice
		var err error
		if cmd.Flag("data").Changed {
			// Use payload file if passed
			handlePayload(cmd, &rfes.RedfishEndpoints)
		} else {
			// ...otherwise use CLI options/args
			rfe := csm.RedfishEndpoint{
				ID:        args[0],
				Name:      args[1],
				IPAddress: args[2],
				MACAddr:   args[3],
			}
			if cmd.Flag("domain").Changed {
				if rfe.Domain, err = cmd.Flags().GetString("domain"); err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch domain")
					logHelpError(cmd)
					os.Exit(1)
				}
			}
			if cmd.Flag("hostname").Changed {
				if rfe.Hostname, err = cmd.Flags().GetString("hostname"); err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch hostname")
					logHelpError(cmd)
					os.Exit(1)
				}
			}
			if cmd.Flag("username").Changed {
				if rfe.User, err = cmd.Flags().GetString("username"); err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch username")
					logHelpError(cmd)
					os.Exit(1)
				}
			}
			if cmd.Flag("password").Changed {
				if rfe.Password, err = cmd.Flags().GetString("password"); err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch password")
					logHelpError(cmd)
					os.Exit(1)
				}
			}
			rfes.RedfishEndpoints = append(rfes.RedfishEndpoints, rfe)
		}

		// Send off request
		_, errs, err := smdClient.PostRedfishEndpoints(rfes, token)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to add redfish endpoint in SMD")
			logHelpError(cmd)
			os.Exit(1)
		}
		// Since smdClient.PostRedfishEndpoints does the addition iteratively, we need to deal with
		// each error that might have occurred.
		var errorsOccurred = false
		for _, err := range errs {
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("SMD redfish endpoint request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to add redfish endpoint(s) to SMD")
				}
				errorsOccurred = true
			}
		}
		if errorsOccurred {
			logHelpError(cmd)
			log.Logger.Warn().Msg("SMD redfish endpoint addition completed with errors")
			os.Exit(1)
		}
	},
}

func init() {
	rfeAddCmd.Flags().String("domain", "", "domain of redfish endpoint's FQDN")
	rfeAddCmd.Flags().String("hostname", "", "hostname of redfish endpoint's FQDN")
	rfeAddCmd.Flags().String("username", "", "username to use when interrogating endpoint")
	rfeAddCmd.Flags().String("password", "", "password to use when interrogating endpoint")
	rfeAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	rfeAddCmd.Flags().VarP(&formatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	rfeAddCmd.RegisterFlagCompletionFunc("format-input", completionFormatData)
	rfeAddCmd.MarkFlagsMutuallyExclusive("domain", "data")
	rfeAddCmd.MarkFlagsMutuallyExclusive("hostname", "data")
	rfeAddCmd.MarkFlagsMutuallyExclusive("username", "data")
	rfeAddCmd.MarkFlagsMutuallyExclusive("password", "data")

	rfeCmd.AddCommand(rfeAddCmd)
}
