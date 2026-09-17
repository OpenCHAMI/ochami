// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package defaults

import (
	metadata_service_client "github.com/openchami/metadata-service/pkg/client"
	"github.com/spf13/cobra"

	api "github.com/openchami/metadata-service/apis/cloud-init.openchami.io/v1"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client/metadata_service"

	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
)

// metadataDefaultsSetOptions holds the flag values for the metadata defaults set command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type metadataDefaultsSetOptions struct {
	Envelope bool
}

// runCoreMetadataDefaultsSet contains the core logic for the metadata defaults set command.
// It takes the parsed options and performs the actual work of setting cluster defaults.
func runCoreMetadataDefaultsSet(cmd *cobra.Command, opts *metadataDefaultsSetOptions, args []string, metadataServiceClient *metadata_service.MetadataServiceClient, rt *cli.Runtime) error {
	// Handle token for this command
	if err := rt.HandleToken(cmd); err != nil {
		return err
	}

	// Determine how to read payload (simple versus advanced API)
	var defaultsSet *api.ClusterDefaults
	var reqErr error
	if opts.Envelope {
		// Use advanced API (spec, metadata, annotations)

		// Read cluster defaults data
		defaults := metadata_service_client.UpdateClusterDefaultsRequest{}
		if cmd.Flag("data").Changed {
			if err := rt.HandlePayload(cmd, &defaults); err != nil {
				return err
			}
		} else {
			if err := rt.HandlePayloadStdin(cmd, &defaults); err != nil {
				return err
			}
		}

		// Send off request
		defaultsSet, reqErr = metadataServiceClient.SetDefaults(cmd.Context(), rt.Token, args[0], defaults)
	} else {
		// Use simple API (spec)

		// Read cluster defaults data
		spec := api.ClusterDefaultsSpec{}
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
		defaultsSet, reqErr = metadataServiceClient.SetDefaultsSpec(cmd.Context(), rt.Token, args[0], spec)
	}
	if reqErr != nil {
		return cli.ClassifyClientError(reqErr, "failed to set cluster defaults", "failed to set cluster defaults")

	}

	// Check that a modified item was returned
	if defaultsSet == nil {
		return cli.Errorf(cli.CodeGeneric, "cluster defaults set returned no resource")
	}

	log.Logger.Debug().Msgf("cluster defaults set: %+v", defaultsSet)

	return nil
}

func newCmdMetadataDefaultsSet() *cobra.Command {
	// metadataDefaultsSetCmd represents the "metadata defaults set" command
	var metadataDefaultsSetCmd = &cobra.Command{
		Use:   "set <uid>",
		Args:  cobra.ExactArgs(1),
		Short: "Set details of an existing cluster defaults spec",
		Long: `Set details of an existing cluster defaults spec.

See ochami-metadata(1) for more details.`,
		Example: `  # Set cluster defaults details using payload data
   ochami metadata defaults set clusterdefaults-d614b918 -d \
     '{
        "base_url": "https://demo.openchami.cluster:8443/cloud-init",
        "cluster_name": "demo",
        "description": "Demo cluster defaults",
        "short_name": "nid",
        "nid_length": 4
      }'

  # Set cluster defaults details preserving labels/annotations (envelope API)
  ochami metadata defaults set clusterdefaults-d614b918 -e -d \
     '{
        "metadata": {
          "labels": {
            "env": "prod"
          }
        },
        "spec": {
          "base_url": "https://demo.openchami.cluster:8443/cloud-init",
          "cluster_name": "demo"
        }
      }'

  # Set cluster defaults details using input payload file
  ochami metadata defaults set clusterdefaults-d614b918 -d @payload.json
  ochami metadata defaults set clusterdefaults-d614b918 -d @payload.yaml -f yaml

  # Set cluster defaults details using data from stdin
  echo '<json_data>' | ochami metadata defaults set clusterdefaults-d614b918 -d @-
  echo '<json_data>' | ochami metadata defaults set clusterdefaults-d614b918
  echo '<yaml_data>' | ochami metadata defaults set clusterdefaults-d614b918 -f yaml -d @-
  echo '<yaml_data>' | ochami metadata defaults set clusterdefaults-d614b918 -f yaml`,
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
			opts := &metadataDefaultsSetOptions{}
			if cmd.Flag("envelope").Changed {
				opts.Envelope, _ = cmd.Flags().GetBool("envelope") //nolint:errcheck // Flag registered with matching type, error impossible
			}

			return runCoreMetadataDefaultsSet(cmd, opts, args, metadataServiceClient, rt)
		},
	}

	// Create flags
	metadataDefaultsSetCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")

	cli.AddFormatInputFlag(metadataDefaultsSetCmd)
	metadataDefaultsSetCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return metadataDefaultsSetCmd
}
