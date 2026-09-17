// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package node

import (
	boot_service_client "github.com/openchami/boot-service/pkg/client"
	"github.com/spf13/cobra"

	api "github.com/openchami/boot-service/apis/boot.openchami.io/v1"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client/boot_service"

	boot_service_lib "github.com/openchami/ochami/internal/cli/boot_service"
)

// bootNodeSetOptions holds the flag values for the boot node set command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type bootNodeSetOptions struct {
	Envelope bool
}

// runCoreBootNodeSet contains the core logic for the boot node set command.
// It takes the parsed options and performs the actual work of setting node details.
func runCoreBootNodeSet(cmd *cobra.Command, opts *bootNodeSetOptions, args []string, bootServiceClient *boot_service.BootServiceClient, rt *cli.Runtime) error {
	// Handle token for this command
	if err := rt.HandleToken(cmd); err != nil {
		return err
	}

	// Determine how to read payload (simple versus advanced API)
	var nodeSet *api.Node
	var reqErr error
	if opts.Envelope {
		// Use advanced API (spec, metadata, annotations)

		// Read node data
		node := boot_service_client.UpdateNodeRequest{}
		if cmd.Flag("data").Changed {
			if err := rt.HandlePayload(cmd, &node); err != nil {
				return err
			}
		} else {
			if err := rt.HandlePayloadStdin(cmd, &node); err != nil {
				return err
			}
		}

		// Send off request
		nodeSet, reqErr = bootServiceClient.SetNode(cmd.Context(), rt.Token, args[0], node)
	} else {
		// Use simple API (spec)

		// Read node data
		spec := api.NodeSpec{}
		if cmd.Flag("data").Changed {
			if err := rt.HandlePayload(cmd, &spec); err != nil {
				return err
			}
		} else {
			if err := rt.HandlePayloadStdin(cmd, &spec); err != nil {
				return err
			}
		}

		// Send off request
		nodeSet, reqErr = bootServiceClient.SetNodeSpec(cmd.Context(), rt.Token, args[0], spec)
	}
	if reqErr != nil {
		return cli.Errorf(cli.CodeNetwork, "failed to set node: %w", reqErr)
	}

	log.Logger.Debug().Msgf("node set: %+v", nodeSet)

	return nil
}

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

  # Set node details preserving labels/annotations (envelope API)
  ochami boot node set nod-bc76f7f2 -e -d \
    '{
       "metadata": {
         "labels": {
           "env": "prod"
         }
       },
       "spec": {
         "xname": "x1000c0s0b0n0",
         "nid": 42
       }
     }'

  # Set node details using input payload file
  ochami boot node set -d @payload.json nod-bc76f7f2
  ochami boot node set -d @payload.yaml -f yaml nod-bc76f7f2

  # Set node details using data from stdin
  echo '<json_data>' | ochami boot node set -d @- nod-bc76f7f2
  echo '<json_data>' | ochami boot node set nod-bc76f7f2
  echo '<yaml_data>' | ochami boot node set -d @- -f yaml nod-bc76f7f2
  echo '<yaml_data>' | ochami boot node set -f yaml nod-bc76f7f2`,
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
			opts := &bootNodeSetOptions{}
			if cmd.Flag("envelope").Changed {
				opts.Envelope, _ = cmd.Flags().GetBool("envelope") //nolint:errcheck // Flag registered with matching type, error impossible
			}

			return runCoreBootNodeSet(cmd, opts, args, bootServiceClient, rt)
		},
	}

	// Create flags
	bootNodeSetCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")

	cli.AddFormatInputFlag(bootNodeSetCmd)
	bootNodeSetCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return bootNodeSetCmd
}
