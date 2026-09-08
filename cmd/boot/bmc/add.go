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
	"github.com/openchami/ochami/pkg/client/boot_service"
)

func newCmdBootBmcAdd() *cobra.Command {
	// bootBmcAddCmd represents the "boot bmc add" command
	var bootBmcAddCmd = &cobra.Command{
		Use:   "add",
		Args:  cobra.NoArgs,
		Short: "Add one or more BMCs to boot-service",
		Long: `Add one or more BMCs to boot-service.

See ochami-boot(1) for more details.`,
		Example: `  # Add BMC using payload data
   ochami boot bmc add -d \
     '{
        "name": "bmc01",
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
          "name": "bmc01",
          "xname": "x1000c0s0b0",
          "description": "Node 1's BMC",
          "interface": {
            "type": "management",
            "mac": "de:ca:fc:0f:fe:e1",
            "ip": "172.16.0.1"
          }
        },
        {
          "name": "bmc02",
          "xname": "x1000c0s0b1",
          "description": "Node 2's BMC",
          "interface": {
            "type": "management",
            "mac": "de:ca:fc:0f:fe:e2",
            "ip": "172.16.0.2"
          }
        }
      ]'

  # Add BMC preserving labels/annotations (envelope API)
  ochami boot bmc add -e -d \
     '{
        "metadata": {
          "name": "x1000c0s0b0",
          "labels": {
            "env": "prod"
          }
        },
        "spec": {
          "xname": "x1000c0s0b0"
        }
      }'

  # Add BMCs using input payload file
  ochami boot bmc add -d @payload.json
  ochami boot bmc add -d @payload.yaml -f yaml

  # Add BMCs using data from stdin
  echo '<json_data>' | ochami boot bmc add -d @-
  echo '<json_data>' | ochami boot bmc add
  echo '<yaml_data>' | ochami boot bmc add -d @- -f yaml`,
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
			envelope, _ := cmd.Flags().GetBool("envelope")

			var bmcsCreated []*api.BMC
			var reqErrs []error
			var reqErr error
			if envelope {
				// Use advanced API (spec, metadata, annotations)

				// Read node data
				bmcs := []boot_service_client.CreateBMCRequest{}
				if cmd.Flag("data").Changed {
					if err := cli.HandlePayloadSlice[boot_service_client.CreateBMCRequest](cmd, &bmcs); err != nil {
						return err
					}
				} else {
					if err := cli.HandlePayloadStdinSlice[boot_service_client.CreateBMCRequest](cmd, &bmcs); err != nil {
						return err
					}
				}

				// Send off requests
				bmcsCreated, reqErrs, reqErr = bootServiceClient.AddBMCs(cli.Token, bmcs)
			} else {
				// Use simple API (spec)

				// Read node data
				bmcs := []boot_service.BMCSpec{}
				if cmd.Flag("data").Changed {
					if err := cli.HandlePayloadSlice[boot_service.BMCSpec](cmd, &bmcs); err != nil {
						return err
					}
				} else {
					if err := cli.HandlePayloadStdinSlice[boot_service.BMCSpec](cmd, &bmcs); err != nil {
						return err
					}
				}

				// Send off requests
				bmcsCreated, reqErrs, reqErr = bootServiceClient.AddBMCSpecs(cli.Token, bmcs)
			}

			// Handle any non-request error
			if reqErr != nil {
				return cli.ClassifyClientError(reqErr, "failed to add BMCs", "failed to add BMCs")

			}

			// Deal with per-request errors
			var reqErrorsOccurred = false
			for _, err := range reqErrs {
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to add BMC")
					reqErrorsOccurred = true
				}
			}
			var names []string
			for _, bmc := range bmcsCreated {
				names = append(names, bmc.Metadata.Name)
			}
			log.Logger.Debug().Msgf("BMCs created: %q", names)
			if reqErrorsOccurred {
				return cli.Errorf(cli.CodeHTTP, "BMC addition completed with errors")
			}

			return nil
		},
	}

	// Create flags
	bootBmcAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	bootBmcAddCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	bootBmcAddCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return bootBmcAddCmd
}
