// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"os"

	boot_service_client "github.com/openchami/boot-service/pkg/client"
	"github.com/spf13/cobra"

	api "github.com/openchami/boot-service/apis/boot.openchami.io/v1"

	"github.com/openchami/ochami/internal/cli"
	boot_service_lib "github.com/openchami/ochami/internal/cli/boot_service"
	"github.com/openchami/ochami/internal/log"
)

func newCmdBootConfigSet() *cobra.Command {
	// bootConfigSetCmd represents the "boot config set" command
	var bootConfigSetCmd = &cobra.Command{
		Use:   "set <uid>",
		Args:  cobra.ExactArgs(1),
		Short: "Set the spec of an existing boot configuration",
		Long: `Set the spec of an existing boot configuration.

See ochami-boot(1) for more details.`,
		Example: `  # Set boot configuration using payload data
  ochami boot config set boo-914afad2 -d \
    '{
       "hosts": [
         "item1",
         "item2"
       ],
       "macs": [
         "de:ca:fc:0f:fe:e1",
         "de:ca:fc:0f:fe:e2"
       ],
       "nids": [
         1,
         2
       ],
       "groups": [
         "group1",
         "group2"
       ],
       "kernel": "http://s3.openchami.cluster/kernels/vmlinuz1",
       "initrd": "http://s3.openchami.cluster/initrds/initramfs1.img",
       "params": "console=tty0,115200n8 console=ttyS0,115200n8",
       "priority": 42
     }'

  # Set boot configuration preserving labels/annotations (envelope API)
  ochami boot config set boo-914afad2 -e -d \
    '{
       "metadata": {
         "labels": {
           "env": "prod"
         }
       },
       "spec": {
         "hosts": ["item1"],
         "kernel": "http://s3.openchami.cluster/kernels/vmlinuz1"
       }
     }'

  # Set boot configuration using input payload file
  ochami boot config set -d @payload.json boo-914afad2
  ochami boot config set -d @payload.yaml -f yaml boo-914afad2

  # Set boot configuration using data from stdin
  echo '<json_data>' | ochami boot config set -d @- boo-914afad2
  echo '<json_data>' | ochami boot config set boo-914afad2
  echo '<yaml_data>' | ochami boot config set -d @- -f yaml boo-914afad2
  echo '<yaml_data>' | ochami boot config set -f yaml boo-914afad2`,
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

			var cfgSet *api.BootConfiguration
			var reqErr error
			if envelope {
				// Use advanced API (spec, metadata, annotations)

				// Read boot configuration data
				bcs := boot_service_client.UpdateBootConfigurationRequest{}
				if cmd.Flag("data").Changed {
					cli.HandlePayload(cmd, &bcs)
				} else {
					cli.HandlePayloadStdin(cmd, &bcs)
				}

				// Send off request
				cfgSet, reqErr = bootServiceClient.SetBootConfig(cli.Token, args[0], bcs)
			} else {
				// Use simple API (spec)

				// Read boot configuration data
				spec := api.BootConfigurationSpec{}
				if cmd.Flag("data").Changed {
					cli.HandlePayload(cmd, &spec)
				} else {
					cli.HandlePayloadStdin(cmd, &spec)
				}

				// Send off request
				cfgSet, reqErr = bootServiceClient.SetBootConfigSpec(cli.Token, args[0], spec)
			}
			if reqErr != nil {
				log.Logger.Error().Err(reqErr).Msg("failed to set boot configuration")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			log.Logger.Debug().Msgf("boot config set: %+v", cfgSet)
		},
	}

	// Create flags
	bootConfigSetCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	bootConfigSetCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	bootConfigSetCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return bootConfigSetCmd
}
