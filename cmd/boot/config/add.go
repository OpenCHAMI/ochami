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
	"github.com/openchami/ochami/pkg/client/boot_service"
)

func newCmdBootConfigAdd() *cobra.Command {
	// bootConfigAddCmd represents the "boot config add" command
	var bootConfigAddCmd = &cobra.Command{
		Use:   "add",
		Args:  cobra.NoArgs,
		Short: "Add new boot configuration(s)",
		Long: `Add new boot configuration(s) for one or more nodes.

See ochami-boot(1) for more details.`,
		Example: `  # Add boot configuration using payload data
  ochami boot config add -d \
    '{
       "name": "compute-boot",
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

  # Add multiple boot configurations using payload data
  ochami boot config add -d \
    '[
       {
         "name": "boot-by-host",
         "hosts": ["host1"],
         "kernel": "http://s3.openchami.cluster/kernels/vmlinuz1",
         "initrd": "http://s3.openchami.cluster/initrds/initramfs1.img",
         "params": "console=tty0,115200n8 console=ttyS0,115200n8",
         "priority": 42
       },
       {
         "name": "boot-by-mac",
         "macs": ["de:ca:fc:0f:fe:ee"],
         "kernel": "http://s3.openchami.cluster/kernels/vmlinuz2",
         "initrd": "http://s3.openchami.cluster/initrds/initramfs2.img",
         "params": "ip=dhcp",
         "priority": 43
       }
     ]'

  # Add boot configuration preserving labels/annotations (envelope API)
  ochami boot config add -e -d \
    '{
       "metadata": {
         "name": "compute-boot",
         "labels": {
           "env": "prod"
         }
       },
       "spec": {
         "hosts": ["item1"],
         "kernel": "http://s3.openchami.cluster/kernels/vmlinuz1"
       }
     }'

  # Add boot configuration using input payload file
  ochami boot config add -d @payload.json
  ochami boot config add -d @payload.yaml -f yaml

  # Add boot configuration using data from stdin
  echo '<json_data>' | ochami boot config add -d @-
  echo '<json_data>' | ochami boot config add
  echo '<yaml_data>' | ochami boot config add -d @- -f yaml
  echo '<yaml_data>' | ochami boot config add -f yaml`,
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

			var cfgsCreated []*api.BootConfiguration
			var reqErrs []error
			var reqErr error
			if envelope {
				// Use advanced API (spec, metadata, annotations)

				// Read boot configuration data
				bcs := []boot_service_client.CreateBootConfigurationRequest{}
				if cmd.Flag("data").Changed {
					if err := cli.HandlePayloadSlice[boot_service_client.CreateBootConfigurationRequest](cmd, &bcs); err != nil {
						return err
					}
				} else {
					if err := cli.HandlePayloadStdinSlice[boot_service_client.CreateBootConfigurationRequest](cmd, &bcs); err != nil {
						return err
					}
				}

				// Send off requests
				cfgsCreated, reqErrs, reqErr = bootServiceClient.AddBootConfigs(cli.Token, bcs)
			} else {
				// Use simple API (spec)

				// Read boot configuration data
				bcs := []boot_service.BootConfigSpec{}
				if cmd.Flag("data").Changed {
					if err := cli.HandlePayloadSlice[boot_service.BootConfigSpec](cmd, &bcs); err != nil {
						return err
					}
				} else {
					if err := cli.HandlePayloadStdinSlice[boot_service.BootConfigSpec](cmd, &bcs); err != nil {
						return err
					}
				}

				// Send off requests
				cfgsCreated, reqErrs, reqErr = bootServiceClient.AddBootConfigSpecs(cli.Token, bcs)
			}

			// Handle any non-request error
			if reqErr != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to add boot configurations: %w", reqErr)
			}

			// Deal with per-request errors
			var reqErrorsOccurred = false
			for _, e := range reqErrs {
				if e != nil {
					log.Logger.Error().Err(e).Msg("failed to add boot configuration")
					reqErrorsOccurred = true
				}
			}
			var names []string
			for _, cfg := range cfgsCreated {
				names = append(names, cfg.Metadata.Name)
			}
			log.Logger.Debug().Msgf("boot configs created: %q", names)
			if reqErrorsOccurred {
				return cli.Errorf(cli.CodeHTTP, "boot configuration addition completed with errors")
			}

			return nil
		},
	}

	// Create flags
	bootConfigAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	bootConfigAddCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	bootConfigAddCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return bootConfigAddCmd
}
