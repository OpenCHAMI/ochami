// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package bmc

import (
	boot_service_client "github.com/openchami/boot-service/pkg/client"
	"github.com/spf13/cobra"

	api "github.com/openchami/boot-service/apis/boot.openchami.io/v1"

	"github.com/openchami/ochami/internal/cli"
	boot_service_lib "github.com/openchami/ochami/internal/cli/boot_service"
	"github.com/openchami/ochami/internal/log"
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

  # Set BMC details preserving labels/annotations (envelope API)
  ochami boot bmc set bmc-773d99bf -e -d \
    '{
       "metadata": {
         "labels": {
           "env": "prod"
         }
       },
       "spec": {
         "xname": "x1000c0s0b0"
       }
     }'

  # Set BMC details using input payload file
  ochami boot bmc set -d @payload.json bmc-773d99bf
  ochami boot bmc set -d @payload.yaml -f yaml bmc-773d99bf

  # Set BMC details using data from stdin
  echo '<json_data>' | ochami boot bmc set -d @- bmc-773d99bf
  echo '<json_data>' | ochami boot bmc set bmc-773d99bf
  echo '<yaml_data>' | ochami boot bmc set -d @- -f yaml bmc-773d99bf
  echo '<yaml_data>' | ochami boot bmc set -f yaml bmc-773d99bf`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			bootServiceClient, err := boot_service_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// Determine how to read payload (simple versus advanced API)
			envelope, flagErr := cmd.Flags().GetBool("envelope")
			if flagErr != nil {
				log.Logger.Warn().Err(flagErr).Msg("failed to read --envelope, falling back to simple API")
			}

			var bmcSet *api.BMC
			var reqErr error
			if envelope {
				// Use advanced API (spec, metadata, annotations)

				// Read BMC data
				bmc := boot_service_client.UpdateBMCRequest{}
				if cmd.Flag("data").Changed {
					if err := cli.HandlePayload(cmd, &bmc); err != nil {
						return err
					}
				} else {
					if err := cli.HandlePayloadStdin(cmd, &bmc); err != nil {
						return err
					}
				}

				// Send off request
				bmcSet, reqErr = bootServiceClient.SetBMC(cli.Token, args[0], bmc)
			} else {
				// Use simple API (spec)

				// Read BMC data
				spec := api.BMCSpec{}
				if cmd.Flag("data").Changed {
					if err := cli.HandlePayload(cmd, &spec); err != nil {
						return err
					}
				} else {
					if err := cli.HandlePayloadStdin(cmd, &spec); err != nil {
						return err
					}
				}

				// Send off request
				bmcSet, reqErr = bootServiceClient.SetBMCSpec(cli.Token, args[0], spec)
			}
			if reqErr != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to set bmc: %w", reqErr)
			}

			log.Logger.Debug().Msgf("bmc set: %+v", bmcSet)

			return nil
		},
	}

	// Create flags
	bootBmcSetCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	bootBmcSetCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	bootBmcSetCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return bootBmcSetCmd
}
