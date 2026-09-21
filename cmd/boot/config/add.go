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
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/boot_service"
)

// bootConfigAddOptions holds the flag values for the boot config add command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type bootConfigAddOptions struct {
	Envelope bool
}

// runCoreBootConfigAdd contains the core logic for the boot config add command.
// It takes the parsed options and performs the actual work of adding boot configurations.
func runCoreBootConfigAdd(cmd *cobra.Command, opts *bootConfigAddOptions, bootServiceClient *boot_service.BootServiceClient, rt *cli.Runtime) error {
	// Handle token for this command
	if err := rt.HandleToken(cmd); err != nil {
		return err
	}

	// Determine how to read payload (simple versus advanced API)
	var results client.BatchResult[*api.BootConfiguration]
	if opts.Envelope {
		// Use advanced API (spec, metadata, annotations)

		// Read boot configuration data
		bcs := []boot_service_client.CreateBootConfigurationRequest{}
		if cmd.Flag("data").Changed {
			if err := cli.HandlePayloadSliceWithRuntime[boot_service_client.CreateBootConfigurationRequest](rt, cmd, &bcs); err != nil {
				return err
			}
		} else {
			if err := cli.HandlePayloadStdinSliceWithRuntime[boot_service_client.CreateBootConfigurationRequest](rt, cmd, &bcs); err != nil {
				return err
			}
		}

		// Send off requests
		results = bootServiceClient.AddBootConfigs(cmd.Context(), rt.Token, bcs)
	} else {
		// Use simple API (spec)

		// Read boot configuration data
		bcs := []boot_service.BootConfigSpec{}
		if cmd.Flag("data").Changed {
			if err := cli.HandlePayloadSliceWithRuntime[boot_service.BootConfigSpec](rt, cmd, &bcs); err != nil {
				return err
			}
		} else {
			if err := cli.HandlePayloadStdinSliceWithRuntime[boot_service.BootConfigSpec](rt, cmd, &bcs); err != nil {
				return err
			}
		}

		// Send off requests
		results = bootServiceClient.AddBootConfigSpecs(cmd.Context(), rt.Token, bcs)
	}

	// Deal with per-request errors
	var reqErrorsOccurred = false
	for _, err := range results.Errors() {
		if err != nil {
			rt.Logger.Error().Err(err).Msg("failed to add boot configuration")
			reqErrorsOccurred = true
		}
	}
	var names []string
	for _, bc := range results.Values() {
		names = append(names, bc.Metadata.Name)
	}
	rt.Logger.Debug().Msgf("boot configurations created: %q", names)
	if reqErrorsOccurred {
		return cli.Errorf(cli.CodeHTTP, "boot configuration addition completed with errors")
	}

	return nil
}

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
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Create client to use for requests
			bootServiceClient, err := boot_service_lib.GetClientWithRuntime(cmd, rt)
			if err != nil {
				return err
			}

			// Extract options from flags
			// Since flags are registered with the correct types on this command,
			// these Get* calls cannot fail, so we ignore errors with explicit comments
			opts := &bootConfigAddOptions{}
			if cmd.Flag("envelope").Changed {
				opts.Envelope, _ = cmd.Flags().GetBool("envelope") //nolint:errcheck // Flag registered with matching type, error impossible
			}

			return runCoreBootConfigAdd(cmd, opts, bootServiceClient, rt)
		},
	}

	// Create flags
	bootConfigAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")

	cli.AddFormatInputFlag(bootConfigAddCmd)
	bootConfigAddCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return bootConfigAddCmd
}
