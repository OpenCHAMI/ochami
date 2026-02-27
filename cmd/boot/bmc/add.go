// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package bmc

import (
	"os"

	boot_service_client "github.com/openchami/boot-service/pkg/client"
	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	boot_service_lib "github.com/OpenCHAMI/ochami/internal/cli/boot_service"
	"github.com/OpenCHAMI/ochami/internal/log"
)

func newCmdBootBmcAdd() *cobra.Command {
	// bootBmcAddCmd represents the "boot bmc add" command
	var bootBmcAddCmd = &cobra.Command{
		Use:   "add",
		Args:  cobra.NoArgs,
		Short: "Add a new BMC to boot-service",
		Long: `Add a new BMC to boot-service.

See ochami-boot(1) for more details.`,
		Example: `  # Add BMC using payload data
  ochami boot bmc add -d \
    '{
      "xname": "x1000c0s0b0",
      "description": "This node's BMC",
      "interface": {
        "type": "management",
        "mac": "de:ca:fc:0f:fe:e1",
        "ip": "172.16.0.254"
      }
    }'

  # Add multiple BMCs using payload data
  ochami boot bmc add -d \
    '[
      {
        "xname": "x1000c0s0b0",
        "description": "Node 1's BMC",
        "interface": {
          "type": "management",
          "mac": "de:ca:fc:0f:fe:e1",
          "ip": "172.16.0.1"
        }
      },
      {
        "xname": "x1000c0s0b1",
        "description": "Node 2's BMC",
        "interface": {
          "type": "management",
          "mac": "de:ca:fc:0f:fe:e2",
          "ip": "172.16.0.2"
        }
      }
    ]'

  # Add BMCs using input payload file
  ochami boot bmc add -d @payload.json
  ochami boot bmc add -d @payload.yaml -f yaml

  # Add BMCs using data from stdin
  echo '<json_data>' | ochami boot bmc add -d @-
  echo '<yaml_data>' | ochami boot bmc add -d @- -f yaml`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			bootServiceClient := boot_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Read node data
			bmcs := []boot_service_client.CreateBMCRequest{}
			cli.HandlePayloadSlice[boot_service_client.CreateBMCRequest](cmd, &bmcs)

			// Send off requests
			bmcsCreated, errs, err := bootServiceClient.AddBMCs(cli.Token, bmcs)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to add BMCs")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Deal with per-request errors
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to add BMC")
					errorsOccurred = true
				}
			}
			log.Logger.Debug().Msgf("BMCs created: %+v", bmcsCreated)
			if errorsOccurred {
				cli.LogHelpError(cmd)
				log.Logger.Warn().Msg("BMC addition completed with errors")
				os.Exit(1)
			}
		},
	}

	// Create flags
	bootBmcAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	bootBmcAddCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	bootBmcAddCmd.MarkFlagsOneRequired("data")

	bootBmcAddCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return bootBmcAddCmd
}
