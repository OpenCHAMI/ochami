// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
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
			envelope, _ := cmd.Flags().GetBool("envelope") //nolint:errcheck // flag is registered with the matching type on this command

			var cfgSet *api.BootConfiguration
			var reqErr error
			if envelope {
				// Use advanced API (spec, metadata, annotations)

				// Read boot configuration data
				bcs := boot_service_client.UpdateBootConfigurationRequest{}
				if cmd.Flag("data").Changed {
					if err := cli.HandlePayload(cmd, &bcs); err != nil {
						return err
					}
				} else {
					if err := cli.HandlePayloadStdin(cmd, &bcs); err != nil {
						return err
					}
				}

				// Send off request
				cfgSet, reqErr = bootServiceClient.SetBootConfig(cli.Token, args[0], bcs)
			} else {
				// Use simple API (spec)

				// Read boot configuration data
				spec := api.BootConfigurationSpec{}
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
				cfgSet, reqErr = bootServiceClient.SetBootConfigSpec(cli.Token, args[0], spec)
			}
			if reqErr != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to set boot configuration: %w", reqErr)
			}

			log.Logger.Debug().Msgf("boot config set: %+v", cfgSet)

			return nil
		},
	}

	// Create flags
	bootConfigSetCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	bootConfigSetCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	bootConfigSetCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return bootConfigSetCmd
}
