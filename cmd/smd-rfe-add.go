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
	Use:   "add -f <payload_file> | (<xname> <name> <ip_addr> <mac_addr>)",
	Args:  cobra.MaximumNArgs(4),
	Short: "Add new redfish endpoint(s)",
	Long: `Add new redfish endpoint(s). An xname, name, IP address, and MAC address are required
unless -f is passed to read from a payload file. Specifying -f also is
mutually exclusive with the other flags of this command and its arguments.
If - is used as the argument to -f, the data is read from standard input.

This command sends a POST to SMD. An access token is required.`,
	Example: `  ochami smd rfe add x3000c1s7b56 bmc-node56 172.16.0.156 de:ca:fc:0f:fe:ee
  ochami smd rfe add -f payload.json
  ochami smd rfe add -f payload.yaml --payload-format yaml
  echo '<json_data>' | ochami smd rfe add -f -
  echo '<yaml_data>' | ochami smd rfe add -f - --payload-format yaml`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Check that all required args are passed
		if len(args) == 0 && !cmd.Flag("payload").Changed {
			printUsageHandleError(cmd)
			os.Exit(0)
		} else if len(args) > 4 {
			return fmt.Errorf("expected 4 arguments (xname, name, ip_addr, mac_addr) but got %d: %v", len(args), args)
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Without a base URI, we cannot do anything
		smdBaseURI, err := getBaseURISMD(cmd)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for SMD")
			os.Exit(1)
		}

		// This endpoint requires authentication, so a token is needed
		setTokenFromEnvVar(cmd)
		checkToken(cmd)

		// Create client to make request to SMD
		smdClient, err := smd.NewClient(smdBaseURI, insecure)
		if err != nil {
			log.Logger.Error().Err(err).Msg("error creating new SMD client")
			os.Exit(1)
		}

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(smdClient.OchamiClient)

		var rfes smd.RedfishEndpointSlice
		if cmd.Flag("payload").Changed {
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
					os.Exit(1)
				}
			}
			if cmd.Flag("hostname").Changed {
				if rfe.Hostname, err = cmd.Flags().GetString("hostname"); err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch hostname")
					os.Exit(1)
				}
			}
			if cmd.Flag("username").Changed {
				if rfe.User, err = cmd.Flags().GetString("username"); err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch username")
					os.Exit(1)
				}
			}
			if cmd.Flag("password").Changed {
				if rfe.Password, err = cmd.Flags().GetString("password"); err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch password")
					os.Exit(1)
				}
			}
			rfes.RedfishEndpoints = append(rfes.RedfishEndpoints, rfe)
		}

		// Send off request
		_, errs, err := smdClient.PostRedfishEndpoints(rfes, token)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to add redfish endpoint in SMD")
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
	rfeAddCmd.Flags().StringP("payload", "f", "", "file containing the request payload; JSON format unless --payload-format specified")
	rfeAddCmd.Flags().StringP("payload-format", "F", defaultPayloadFormat, "format of payload file (yaml,json) passed with --payload")

	rfeAddCmd.MarkFlagsMutuallyExclusive("domain", "payload")
	rfeAddCmd.MarkFlagsMutuallyExclusive("hostname", "payload")
	rfeAddCmd.MarkFlagsMutuallyExclusive("username", "payload")
	rfeAddCmd.MarkFlagsMutuallyExclusive("password", "payload")

	rfeCmd.AddCommand(rfeAddCmd)
}
