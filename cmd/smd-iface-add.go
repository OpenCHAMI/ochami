// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/smd"
	"github.com/spf13/cobra"
)

// ifaceAddCmd represents the smd-iface-add command
var ifaceAddCmd = &cobra.Command{
	Use:   "add (-d (<payload_data> | @<payload_file>)) | (<comp_id> <mac_addr> (<net_name>,<ip_addr>)...)",
	Short: "Add new ethernet interface(s)",
	Long: `Add new ethernet interface(s). A component ID (usually an xname), MAC address, and
one or more pairs of network name and IP address (delimited by a comma)
are required. Alternatively, pass -d to pass raw payload data
or (if flag argument starts with @) a file containing the
payload data. -f can be specified to change the format of
the input payload data ('json' by default), but the rules
above still apply for the payload. If "-" is used as the
input payload filename, the data is read from standard input.

This command sends a POST to SMD. An access token is required.

See ochami-smd(1) for more details.`,
	Example: `  # Add ethernet interface using CLI flags
  ochami smd iface add x3000c1s7b55n0 de:ca:fc:0f:fe:ee NMN,172.16.0.55
  ochami smd iface add -d "Node Management for n55" x3000c1s7b55n0 de:ca:fc:0f:fe:ee NMN,172.16.0.55
  ochami smd iface add x3000c1s7b55n0 de:ca:fc:0f:fe:ee external,10.1.0.55 internal,172.16.0.55

  # Add ethernet interfaces using input payload file
  ochami smd iface add -d @payload.json
  ochami smd iface add -d @payload.yaml -f yaml

  # Add ethernet interfaces using data from standard input
  echo '<json_data>' | ochami smd iface add -d @-
  echo '<yaml_data>' | ochami smd iface add -d @- -f yaml`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Check that all required args are passed
		if len(args) == 0 && !cmd.Flag("data").Changed {
			printUsageHandleError(cmd)
			os.Exit(0)
		} else if len(args) < 3 {
			return fmt.Errorf("expected at least 3 arguments (comp_id, mac_addr, net_ip_paor) but got %d: %v", len(args), args)
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Without a base URI, we cannot do anything
		smdBaseURI, err := getBaseURISMD(cmd)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for SMD")
			logHelpError(cmd)
			os.Exit(1)
		}

		// This endpoint requires authentication, so a token is needed
		setTokenFromEnvVar(cmd)
		checkToken(cmd)

		// Create client to make request to SMD
		smdClient, err := smd.NewClient(smdBaseURI, insecure)
		if err != nil {
			log.Logger.Error().Err(err).Msg("error creating new SMD client")
			logHelpError(cmd)
			os.Exit(1)
		}

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(smdClient.OchamiClient)

		var eis []smd.EthernetInterface
		if cmd.Flag("data").Changed {
			// Use payload file if passed
			handlePayload(cmd, &eis)
		} else {
			// ...otherwise use CLI options/args
			var nets []smd.EthernetIP
			for i := 2; i < len(args); i++ {
				tokens := strings.SplitN(args[i], ",", 2)
				if ip := net.ParseIP(tokens[1]); ip.To4() == nil {
					log.Logger.Error().Msgf("invalid IP address: %s", tokens[1])
					os.Exit(1)
				}
				net := smd.EthernetIP{
					Network:   tokens[0],
					IPAddress: tokens[1],
				}
				nets = append(nets, net)
			}
			ei := smd.EthernetInterface{
				ComponentID: args[0],
				Description: cmd.Flag("description").Value.String(),
				MACAddress:  args[1],
				IPAddresses: nets,
			}
			eis = append(eis, ei)
		}

		// Send off request
		_, errs, err := smdClient.PostEthernetInterfaces(eis, token)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to add ethernet interface in SMD")
			logHelpError(cmd)
			os.Exit(1)
		}
		// Since smdClient.PostEthernetInterfaces does the addition iteratively, we need to deal with
		// each error that might have occurred.
		var errorsOccurred = false
		for _, err := range errs {
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("SMD ethernet interface request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to add ethernet interfaces to SMD")
				}
				errorsOccurred = true
			}
		}
		if errorsOccurred {
			logHelpError(cmd)
			log.Logger.Warn().Msg("SMD ethernet interface addition completed with errors")
			os.Exit(1)
		}
	},
}

func init() {
	ifaceAddCmd.Flags().StringP("description", "D", "Undescribed Ethernet Interface", "description of interface")
	ifaceAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	ifaceAddCmd.Flags().StringP("format-input", "f", defaultInputFormat, "format of input payload data (json,yaml)")

	ifaceAddCmd.MarkFlagsMutuallyExclusive("description", "data")

	ifaceCmd.AddCommand(ifaceAddCmd)
}
