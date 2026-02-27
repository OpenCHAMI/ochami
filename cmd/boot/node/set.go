// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package node

import (
	"os"

	boot_service_client "github.com/openchami/boot-service/pkg/client"
	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	boot_service_lib "github.com/OpenCHAMI/ochami/internal/cli/boot_service"
	"github.com/OpenCHAMI/ochami/internal/log"
)

func newCmdBootNodeSet() *cobra.Command {
	// bootNodeSetCmd represents the "boot node set" command
	var bootNodeSetCmd = &cobra.Command{
		Use:   "set <uid>",
		Args:  cobra.ExactArgs(1),
		Short: "Set details of an existing node",
		Long: `Set details of an existing node configuration.

See ochami-boot(1) for more details.`,
		Example: `  # Set node details using payload data
  ochami boot node set nod-bc76f7f2 -d \
    '{
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

  # Set boot configuration using input payload file
  ochami boot node set -d @payload.json nod-bc76f7f2
  ochami boot node set -d @payload.yaml -f yaml nod-bc76f7f2

  # Set boot configuration using data from stdin
  echo '<json_data>' | ochami boot node set -d @- nod-bc76f7f2
  echo '<yaml_data>' | ochami boot node set -d @- -f yaml nod-bc76f7f2`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			bootServiceClient := boot_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Read node data
			node := boot_service_client.UpdateNodeRequest{}
			cli.HandlePayload(cmd, &node)

			// Send off requests
			nodeSet, err := bootServiceClient.SetNode(cli.Token, args[0], node)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to set node")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			log.Logger.Debug().Msgf("node set: %+v", nodeSet)
		},
	}

	// Create flags
	bootNodeSetCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	bootNodeSetCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	bootNodeSetCmd.MarkFlagsOneRequired("data")

	bootNodeSetCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return bootNodeSetCmd
}
