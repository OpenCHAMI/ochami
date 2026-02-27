// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"os"

	boot_service_client "github.com/openchami/boot-service/pkg/client"
	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	boot_service_lib "github.com/OpenCHAMI/ochami/internal/cli/boot_service"
	"github.com/OpenCHAMI/ochami/internal/log"
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

  # Set boot configuration using input payload file
  ochami boot config set -d @payload.json boo-914afad2
  ochami boot config set -d @payload.yaml -f yaml boo-914afad2

  # Set boot configuration using data from stdin
  echo '<json_data>' | ochami boot config set -d @- boo-914afad2
  echo '<yaml_data>' | ochami boot config set -d @- -f yaml boo-914afad2`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			bootServiceClient := boot_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Read boot configuration data
			bcs := boot_service_client.UpdateBootConfigurationRequest{}
			cli.HandlePayload(cmd, &bcs)

			// Send off requests
			cfgSet, err := bootServiceClient.SetBootConfig(cli.Token, args[0], bcs)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to set boot configuration")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			log.Logger.Debug().Msgf("boot config set: %+v", cfgSet)
		},
	}

	// Create flags
	bootConfigSetCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	bootConfigSetCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	bootConfigSetCmd.MarkFlagsOneRequired("data")

	bootConfigSetCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return bootConfigSetCmd
}
