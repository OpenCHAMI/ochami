// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package defaults

import (
	metadata_service_client "github.com/openchami/metadata-service/pkg/client"
	"github.com/spf13/cobra"

	api "github.com/openchami/metadata-service/apis/cloud-init.openchami.io/v1"

	"github.com/openchami/ochami/internal/cli"
	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/metadata_service"
)

// metadataDefaultsAddOptions holds the flag values for the metadata defaults add command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type metadataDefaultsAddOptions struct {
	Envelope bool
}

func runCoreMetadataDefaultsAddWithRuntime(cmd *cobra.Command, opts *metadataDefaultsAddOptions, metadataServiceClient *metadata_service.MetadataServiceClient, rt *cli.Runtime) error {
	// Handle token for this command
	if err := rt.HandleToken(cmd); err != nil {
		return err
	}

	// Determine how to read payload (simple versus advanced API)
	var results client.BatchResult[api.ClusterDefaults]
	if opts.Envelope {
		// Use advanced API (spec, metadata, annotations)

		// Read cluster defaults data
		defaults := []metadata_service_client.CreateClusterDefaultsRequest{}
		if cmd.Flag("data").Changed {
			if err := cli.HandlePayloadSliceWithRuntime[metadata_service_client.CreateClusterDefaultsRequest](rt, cmd, &defaults); err != nil {
				return err
			}
		} else {
			if err := cli.HandlePayloadStdinSliceWithRuntime[metadata_service_client.CreateClusterDefaultsRequest](rt, cmd, &defaults); err != nil {
				return err
			}
		}

		// Send off requests
		results = metadataServiceClient.AddDefaults(cmd.Context(), rt.Token, defaults)
	} else {
		// Use simple API (spec)

		// Read cluster defaults data
		defaults := []metadata_service.ClusterDefaultsSpec{}
		if cmd.Flag("data").Changed {
			if err := cli.HandlePayloadSliceWithRuntime[metadata_service.ClusterDefaultsSpec](rt, cmd, &defaults); err != nil {
				return err
			}
		} else {
			if err := cli.HandlePayloadStdinSliceWithRuntime[metadata_service.ClusterDefaultsSpec](rt, cmd, &defaults); err != nil {
				return err
			}
		}

		// Send off requests
		results = metadataServiceClient.AddDefaultsSpecs(cmd.Context(), rt.Token, defaults)
	}

	// Deal with per-request errors
	var reqErrorsOccurred = false
	for _, err := range results.Errors() {
		if err != nil {
			rt.Logger.Error().Err(err).Msg("failed to add cluster defaults")
			reqErrorsOccurred = true
		}
	}

	// Print names of created items
	var names []string
	for _, defaults := range results.Values() {
		names = append(names, defaults.Metadata.Name)
	}
	rt.Logger.Info().Msgf("Cluster defaults created: %q", names)

	// Warn if any request errors occurred
	if reqErrorsOccurred {
		return cli.Errorf(cli.CodeHTTP, "Cluster defaults addition completed with errors")
	}

	return nil
}

func newCmdMetadataDefaultsAdd() *cobra.Command {
	// metadataDefaultsAddCmd represents the "metadata defaults add" command
	var metadataDefaultsAddCmd = &cobra.Command{
		Use:   "add",
		Args:  cobra.NoArgs,
		Short: "Add one or more cluster defaults to metadata-service",
		Long: `Add one or more cluster defaults to metadata-service.

See ochami-metadata(1) for more details.`,
		Example: `  # Add cluster defaults using payload data
  ochami metadata defaults add -d \
    '{
       "name": "demo-cluster-defaults",
       "base_url": "https://demo.openchami.cluster:8443/cloud-init",
       "cluster_name": "demo",
       "description": "Demo cluster defaults",
       "short_name": "nid",
       "nid_length": 4
     }'

  # Add multiple cluster defaults using payload data
  ochami metadata defaults add -d \
    '[
       {
         "name": "demo1-cluster-defaults",
         "base_url": "https://demo1.openchami.cluster:8443/cloud-init",
         "cluster_name": "demo1",
         "description": "Demo 1 cluster defaults",
         "short_name": "nid",
         "nid_length": 4
       },
       {
         "name": "demo2-cluster-defaults",
         "base_url": "https://demo2.openchami.cluster:8443/cloud-init",
         "cluster_name": "demo2",
         "description": "Demo 2 cluster defaults",
         "short_name": "de",
         "nid_length": 3
       }
     ]'

  # Add multiple cluster defaults using YAML array of specs
  ochami metadata defaults add -f yaml <<'EOF'
   - name: demo1-cluster-defaults
     base_url: "https://demo1.openchami.cluster:8443/cloud-init"
     cluster_name: "demo1"
   - name: demo2-cluster-defaults
     base_url: "https://demo2.openchami.cluster:8443/cloud-init"
     cluster_name: "demo2"
   EOF

  # Add cluster defaults preserving labels/annotations (envelope API)
  ochami metadata defaults add -e -d \
    '{
       "metadata": {
         "name": "demo-cluster-defaults",
         "labels": {
           "env": "prod"
         }
       },
       "spec": {
         "base_url": "https://demo.openchami.cluster:8443/cloud-init",
         "cluster_name": "demo"
       }
     }'

  # Add cluster defaults using input payload file
  ochami metadata defaults add -d @payload.json
  ochami metadata defaults add -d @payload.yaml -f yaml

  # Add cluster defaults using data from stdin
  echo '<json_data>' | ochami metadata defaults add -d @-
  echo '<json_data>' | ochami metadata defaults add
  echo '<yaml_data>' | ochami metadata defaults add -f yaml -d @-
  echo '<yaml_data>' | ochami metadata defaults add -f yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Create client to use for requests
			metadataServiceClient, err := metadata_service_lib.GetClientWithRuntime(cmd, rt)
			if err != nil {
				return err
			}

			// Extract options from flags
			// Since flags are registered with the correct types on this command,
			// these Get* calls cannot fail, so we ignore errors with explicit comments
			opts := &metadataDefaultsAddOptions{}
			if cmd.Flag("envelope").Changed {
				opts.Envelope, _ = cmd.Flags().GetBool("envelope") //nolint:errcheck // Flag registered with matching type, error impossible
			}

			return runCoreMetadataDefaultsAddWithRuntime(cmd, opts, metadataServiceClient, rt)
		},
	}

	// Create flags
	metadataDefaultsAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")

	cli.AddFormatInputFlag(metadataDefaultsAddCmd)
	metadataDefaultsAddCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return metadataDefaultsAddCmd
}
