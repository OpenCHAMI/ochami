// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package node

import (
	"os"

	boot_service_client "github.com/openchami/boot-service/pkg/client"
	"github.com/spf13/cobra"

	api "github.com/openchami/boot-service/apis/boot.openchami.io/v1"

	"github.com/openchami/ochami/internal/cli"
	boot_service_lib "github.com/openchami/ochami/internal/cli/boot_service"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client/boot_service"
)

func newCmdBootNodeAdd() *cobra.Command {
	// bootNodeAddCmd represents the "boot node add" command
	var bootNodeAddCmd = &cobra.Command{
		Use:   "add",
		Args:  cobra.NoArgs,
		Short: "Add a new node to boot-service",
		Long: `Add a new node to boot-service.

See ochami-boot(1) for more details.`,
		Example: `  # Add node using payload data
  ochami boot node add -d \
    '{
       "name": "node01",
       "xname": "x1000c0s0b0n0",
       "nid": 42,
       "bootMac": "de:ca:fc:0f:fe:e1",
       "role": "example-role",
       "subRole": "example-subrole",
       "hostname": "ex01.example.org",
       "interfaces": [
         {
           "type": "management",
           "mac": "de:ca:fc:0f:fe:e1",
           "ip": "172.16.0.1"
         }
       ],
       "groups": [
         "group1",
         "group2"
       ]
     }'

  # Add multiple nodes using payload data
  ochami boot node add -d \
    '[
       {
         "name": "node01",
         "xname": "x1000c0s0b0n0",
         "nid": 42,
         "bootMac": "de:ca:fc:0f:fe:e1",
         "hostname": "ex01.example.org",
         "interfaces": [
           {
             "type": "management",
             "mac": "de:ca:fc:0f:fe:e1",
             "ip": "172.16.0.1"
           }
         ]
       },
       {
         "name": "node02",
         "xname": "x1000c0s0b0n1",
         "nid": 43,
         "bootMac": "de:ca:fc:0f:fe:e2",
         "hostname": "ex02.example.org",
         "interfaces": [
           {
             "type": "management",
             "mac": "de:ca:fc:0f:fe:e2",
             "ip": "172.16.0.2"
           }
         ]
       }
     ]'

  # Add node preserving labels/annotations (envelope API)
  ochami boot node add -e -d \
    '{
       "metadata": {
         "name": "x1000c0s0b0n0",
         "labels": {
           "env": "prod"
         }
       },
       "spec": {
         "xname": "x1000c0s0b0n0",
         "nid": 42
       }
     }'

  # Add nodes using input payload file
  ochami boot node add -d @payload.json
  ochami boot node add -d @payload.yaml -f yaml

  # Add nodes using data from stdin
  echo '<json_data>' | ochami boot node add -d @-
  echo '<json_data>' | ochami boot node add
  echo '<yaml_data>' | ochami boot node add -d @- -f yaml
  echo '<yaml_data>' | ochami boot node add -f yaml`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			bootServiceClient := boot_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Determine how to read payload (simple versus advanced API)
			envelope, flagErr := cmd.Flags().GetBool("envelope")
			if flagErr != nil {
				log.Logger.Warn().Err(flagErr).Msg("failed to read --envelope, falling back to simple API")
			}

			var nodesCreated []*api.Node
			var reqErrs []error
			var reqErr error
			if envelope {
				// Use advanced API (spec, metadata, annotations)

				// Read node data
				nodes := []boot_service_client.CreateNodeRequest{}
				if cmd.Flag("data").Changed {
					cli.HandlePayloadSlice[boot_service_client.CreateNodeRequest](cmd, &nodes)
				} else {
					cli.HandlePayloadStdinSlice[boot_service_client.CreateNodeRequest](cmd, &nodes)
				}

				// Send off requests
				nodesCreated, reqErrs, reqErr = bootServiceClient.AddNodes(cli.Token, nodes)
			} else {
				// Use simple API (spec)

				// Read node data
				nodes := []boot_service.NodeSpec{}
				if cmd.Flag("data").Changed {
					cli.HandlePayloadSlice[boot_service.NodeSpec](cmd, &nodes)
				} else {
					cli.HandlePayloadStdinSlice[boot_service.NodeSpec](cmd, &nodes)
				}

				// Send off requests
				nodesCreated, reqErrs, reqErr = bootServiceClient.AddNodeSpecs(cli.Token, nodes)
			}

			// Handle any non-request error
			if reqErr != nil {
				log.Logger.Error().Err(reqErr).Msg("failed to add nodes")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Deal with per-request errors
			var reqErrorsOccurred = false
			for _, err := range reqErrs {
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to add node")
					reqErrorsOccurred = true
				}
			}
			var names []string
			for _, node := range nodesCreated {
				names = append(names, node.Metadata.Name)
			}
			log.Logger.Debug().Msgf("nodes created: %q", names)
			if reqErrorsOccurred {
				cli.LogHelpError(cmd)
				log.Logger.Warn().Msg("node addition completed with errors")
				os.Exit(1)
			}
		},
	}

	// Create flags
	bootNodeAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	bootNodeAddCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	bootNodeAddCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return bootNodeAddCmd
}
