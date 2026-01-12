// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package iface

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/smd"

	smd_lib "github.com/OpenCHAMI/ochami/internal/cli/smd"
)

func newCmdIfaceAdd() *cobra.Command {
	// ifaceAddCmd represents the "smd iface add" command
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

This command sends a POST to SMD. An access cli.Token is required.

See ochami-smd(1) for more details.`,
		Example: `  # Add ethernet interface using CLI flags
  ochami smd iface add x3000c1s7b55n0 de:ca:fc:0f:fe:ee NMN,172.16.0.55
  ochami smd iface add -D "Node Management for n55" x3000c1s7b55n0 de:ca:fc:0f:fe:ee NMN,172.16.0.55
  ochami smd iface add x3000c1s7b55n0 de:ca:fc:0f:fe:ee external,10.1.0.55 internal,172.16.0.55

  # Add ethernet interfaces using input payload file
  ochami smd iface add -d @payload.json
  ochami smd iface add -d @payload.yaml -f yaml

  # Add ethernet interfaces using data from standard input
  echo '<json_data>' | ochami smd iface add -d @-
  echo '<yaml_data>' | ochami smd iface add -d @- -f yaml`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Check that all required args are passed
			if !cmd.Flag("data").Changed {
				if len(args) != 3 {
					return fmt.Errorf("expected -d or >= 3 arguments (component id, mac address, network name, ip address), got %d", len(args))
				}
			} else {
				if len(args) > 0 {
					log.Logger.Warn().Msgf("raw data passed, ignoring extra arguments: %v", args)
				}
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			smdClient := smd_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			var eis []smd.EthernetInterface
			if cmd.Flag("data").Changed {
				// Use payload file if passed
				cli.HandlePayload(cmd, &eis)
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
			_, errs, err := smdClient.PostEthernetInterfaces(eis, cli.Token)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to add ethernet interface in SMD")
				cli.LogHelpError(cmd)
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
				cli.LogHelpError(cmd)
				log.Logger.Warn().Msg("SMD ethernet interface addition completed with errors")
				os.Exit(1)
			}
		},
	}

	// Create flags
	ifaceAddCmd.Flags().StringP("description", "D", "Undescribed Ethernet Interface", "description of interface")
	ifaceAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	ifaceAddCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	ifaceAddCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)
	ifaceAddCmd.MarkFlagsMutuallyExclusive("description", "data")

	return ifaceAddCmd
}
