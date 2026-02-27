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

func newCmdBootBmcSet() *cobra.Command {
	// bootBmcSetCmd represents the "boot bmc set" command
	var bootBmcSetCmd = &cobra.Command{
		Use:   "set <uid>",
		Args:  cobra.ExactArgs(1),
		Short: "Set details of an existing BMC",
		Long: `Set details of an existing BMC spec.

See ochami-boot(1) for more details.`,
		Example: `  # Set BMC details using payload data
  ochami boot bmc set bmc-773d99bf -d \
    '{
      "xname": "x1000c0s0b0",
      "description": "This node's BMC",
      "interface": {
        "type": "management",
        "mac": "de:ca:fc:0f:fe:e1",
        "ip": "172.16.0.254"
      }
    }'

  # Set BMC details using input payload file
  ochami boot bmc set -d @payload.json bmc-773d99bf
  ochami boot bmc set -d @payload.yaml -f yaml bmc-773d99bf

  # Set BMC details using data from stdin
  echo '<json_data>' | ochami boot bmc set -d @- bmc-773d99bf
  echo '<yaml_data>' | ochami boot bmc set -d @- -f yaml bmc-773d99bf`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			bootServiceClient := boot_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Read BMC data
			bmc := boot_service_client.UpdateBMCRequest{}
			cli.HandlePayload(cmd, &bmc)

			// Send off requests
			bmcSet, err := bootServiceClient.SetBMC(cli.Token, args[0], bmc)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to set bmc")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			log.Logger.Debug().Msgf("bmc set: %+v", bmcSet)
		},
	}

	// Create flags
	bootBmcSetCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	bootBmcSetCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	bootBmcSetCmd.MarkFlagsOneRequired("data")

	bootBmcSetCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return bootBmcSetCmd
}
