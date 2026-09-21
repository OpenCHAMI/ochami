// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package node

import (
	api "github.com/openchami/boot-service/apis/boot.openchami.io/v1"
	boot_service_client "github.com/openchami/boot-service/pkg/client"
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/boot_service"

	boot_service_lib "github.com/openchami/ochami/internal/cli/boot_service"
)

// bootNodeAddOptions holds the flag values for the boot node add command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type bootNodeAddOptions struct {
	Envelope bool
}

// runCoreBootNodeAdd contains the core logic for the boot node add command.
// It takes the parsed options and performs the actual work of adding nodes.
func runCoreBootNodeAdd(cmd *cobra.Command, opts *bootNodeAddOptions, bootServiceClient *boot_service.BootServiceClient, rt *cli.Runtime) error {
	// Handle token for this command
	if err := rt.HandleToken(cmd); err != nil {
		return err
	}

	// Determine how to read payload (simple versus advanced API)
	var results client.BatchResult[*api.Node]
	if opts.Envelope {
		// Use advanced API (spec, metadata, annotations)

		// Read node data
		nodes := []boot_service_client.CreateNodeRequest{}
		if cmd.Flag("data").Changed {
			if err := cli.HandlePayloadSliceWithRuntime[boot_service_client.CreateNodeRequest](rt, cmd, &nodes); err != nil {
				return err
			}
		} else {
			if err := cli.HandlePayloadStdinSliceWithRuntime[boot_service_client.CreateNodeRequest](rt, cmd, &nodes); err != nil {
				return err
			}
		}

		// Send off requests
		results = bootServiceClient.AddNodes(cmd.Context(), rt.Token, nodes)
	} else {
		// Use simple API (spec)

		// Read node data
		nodes := []boot_service.NodeSpec{}
		if cmd.Flag("data").Changed {
			if err := cli.HandlePayloadSliceWithRuntime[boot_service.NodeSpec](rt, cmd, &nodes); err != nil {
				return err
			}
		} else {
			if err := cli.HandlePayloadStdinSliceWithRuntime[boot_service.NodeSpec](rt, cmd, &nodes); err != nil {
				return err
			}
		}

		// Send off requests
		results = bootServiceClient.AddNodeSpecs(cmd.Context(), rt.Token, nodes)
	}

	// Deal with per-request errors
	var reqErrorsOccurred = false
	for _, e := range results.Errors() {
		if e != nil {
			rt.Logger.Error().Err(e).Msg("failed to add node")
			reqErrorsOccurred = true
		}
	}
	var names []string
	for _, node := range results.Values() {
		names = append(names, node.Metadata.Name)
	}
	rt.Logger.Debug().Msgf("nodes created: %q", names)
	if reqErrorsOccurred {
		return cli.Errorf(cli.CodeHTTP, "node addition completed with errors")
	}

	return nil
}

func newCmdBootNodeAdd() *cobra.Command {
	// bootNodeAddCmd represents the "boot node add" command
	var bootNodeAddCmd = &cobra.Command{
		Use:   "add",
		Args:  cobra.NoArgs,
		Short: "Add a new node to boot-service",
		Long: `Add a new node to boot-service.

See ochami-boot(1) for more details.`,
		Example: `  # Add node using payload data
  ochami boot node add -d \
    '{
       "name": "node01",
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

  # Add multiple nodes using payload data
  ochami boot node add -d \
    '[
       {
         "name": "node01",
         "xname": "x1000c0s0b0n0",
         "nid": 42,
         "bootMac": "de:ca:fc:0f:fe:e1",
         "hostname": "ex01.example.org",
         "interfaces": [
           {
             "type": "management",
             "mac": "de:ca:fc:0f:fe:e1",
             "ip": "172.16.0.1"
           }
         ]
       },
       {
         "name": "node02",
         "xname": "x1000c0s0b0n1",
         "nid": 43,
         "bootMac": "de:ca:fc:0f:fe:e2",
         "hostname": "ex02.example.org",
         "interfaces": [
           {
             "type": "management",
             "mac": "de:ca:fc:0f:fe:e2",
             "ip": "172.16.0.2"
           }
         ]
       }
     ]'

  # Add node preserving labels/annotations (envelope API)
  ochami boot node add -e -d \
    '{
       "metadata": {
         "name": "x1000c0s0b0n0",
         "labels": {
           "env": "prod"
         }
       },
       "spec": {
         "xname": "x1000c0s0b0n0",
         "nid": 42
       }
     }'

  # Add nodes using input payload file
  ochami boot node add -d @payload.json
  ochami boot node add -d @payload.yaml -f yaml

  # Add nodes using data from stdin
  echo '<json_data>' | ochami boot node add -d @-
  echo '<json_data>' | ochami boot node add
  echo '<yaml_data>' | ochami boot node add -d @- -f yaml
  echo '<yaml_data>' | ochami boot node add -f yaml`,
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
			opts := &bootNodeAddOptions{}
			if cmd.Flag("envelope").Changed {
				opts.Envelope, _ = cmd.Flags().GetBool("envelope") //nolint:errcheck // Flag registered with matching type, error impossible
			}

			return runCoreBootNodeAdd(cmd, opts, bootServiceClient, rt)
		},
	}

	// Create flags
	bootNodeAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")

	cli.AddFormatInputFlag(bootNodeAddCmd)
	bootNodeAddCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return bootNodeAddCmd
}
