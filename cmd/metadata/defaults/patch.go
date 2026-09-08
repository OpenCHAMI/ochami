// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package defaults

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"
)

func newCmdMetadataDefaultsPatch() *cobra.Command {
	formatPatch := client.PatchMethodRFC7386
	var setList, unsetList, addList, removeList []string
	// metadataDefaultsPatchCmd represents the "metadata defaults patch" command
	var metadataDefaultsPatchCmd = &cobra.Command{
		Use:   "patch <uid>",
		Args:  cobra.ExactArgs(1),
		Short: "Patch an existing cluster defaults spec",
		Long: `Patch an existing cluster defaults spec using various patch formats.

IMPORTANT: Only the spec portion of the resource can be patched.
Metadata (name, labels, annotations) and status are managed by the API.
Attempts to patch metadata or status fields will be ignored.

If --set/--unset/--add/--remove are specified or --patch-method is 'keyval',
the manual, key-value patch method using dot notation (e.g. key.subkey=value)
is used.

Otherwise, stdin and/or --data can be used to pass in raw patch data, using
--patch-format to specify the patch format (see examples below).

--format-input can only be used with stdin/--data. It can be used to tell
ochami to use a different format (e.g. YAML) for the data input for either
of these methods.

See ochami-metadata(1) for more details.`,
		Example: `  # Patch using JSON patch (RFC 6902)
   ochami metadata defaults patch clusterdefaults-d614b918 --patch-method rfc6902 --data '[
     {"op":"replace","path":"/base_url","value":"https://demo.openchami.cluster:8443/metadata-service"},
     {"op":"replace","path":"/short_name","value":"de"}
   ]'

  # Patch specific fields using JSON merge patch (RFC 7386) (simple merge)
  ochami metadata defaults patch clusterdefaults-d614b918 --patch-method rfc7386 --data '{"short_name":"de"}'

  # Patch specific fields using dot notation for keys (shorthand patch)
  ochami metadata defaults patch clusterdefaults-d614b918 --patch-method keyval --set short_name='de'

  # Patch using payload file
  ochami metadata defaults patch clusterdefaults-d614b918 -d @payload.json
  ochami metadata defaults patch clusterdefaults-d614b918 -d @payload.yaml -f yaml

  # Patch using stdin
  echo '<json_data>' | ochami metadata defaults patch clusterdefaults-d614b918 -d @-
  echo '<json_data>' | ochami metadata defaults patch clusterdefaults-d614b918
  echo '<yaml_data>' | ochami metadata defaults patch clusterdefaults-d614b918 -f yaml -d @-
  echo '<yaml_data>' | ochami metadata defaults patch clusterdefaults-d614b918 -f yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			metadataServiceClient, err := metadata_service_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			var patchData interface{}
			if cmd.Flag("set").Changed || cmd.Flag("unset").Changed || cmd.Flag("add").Changed || cmd.Flag("remove").Changed {
				if cmd.Flag("patch-method").Changed && formatPatch != client.PatchMethodKeyVal {
					log.Logger.Warn().Msg("overriding --patch-method since --set/--unset/--add/--remove was passed")
				}

				newPatchMethod, pd, err := client.NewKeyValPatchData(setList, unsetList, addList, removeList)
				if err != nil {
					return err
				}
				formatPatch = newPatchMethod
				patchData = pd
			} else {
				if cmd.Flag("data").Changed {
					if err := cli.HandlePayload(cmd, &patchData); err != nil {
						return err
					}
				} else {
					if err := cli.HandlePayloadStdin(cmd, &patchData); err != nil {
						return err
					}
				}
			}

			defaultsPatched, err := metadataServiceClient.PatchDefaults(cli.Token, formatPatch, args[0], patchData)
			if err != nil {
				return cli.ClassifyClientError(err, "failed to patch cluster defaults", "failed to patch cluster defaults")

			}

			// Check that a modified item was returned
			if defaultsPatched == nil {
				return cli.Errorf(cli.CodeGeneric, "cluster defaults patch returned no resource")
			}

			// Print UIDs of modified items
			log.Logger.Info().Msgf("Cluster defaults patched: %+v", []string{defaultsPatched.Metadata.UID})

			return nil
		},
	}

	// Create flags
	metadataDefaultsPatchCmd.Flags().StringArrayVar(&setList, "set", nil, "set field value using dot notation (field=value)")
	metadataDefaultsPatchCmd.Flags().StringArrayVar(&unsetList, "unset", nil, "unset field using dot notation")
	metadataDefaultsPatchCmd.Flags().StringArrayVar(&addList, "add", nil, "add value to array field (field=value)")
	metadataDefaultsPatchCmd.Flags().StringArrayVar(&removeList, "remove", nil, "remove value from array field (field=value)")
	metadataDefaultsPatchCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	metadataDefaultsPatchCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data for JSON patch formats (json,json-pretty,yaml)")
	metadataDefaultsPatchCmd.Flags().VarP(&formatPatch, "patch-method", "p", "type of patch to use (rfc6902,rfc7386,keyval)")

	for _, flag := range []string{"set", "unset", "add", "remove"} {
		metadataDefaultsPatchCmd.MarkFlagsMutuallyExclusive("format-input", flag)
		metadataDefaultsPatchCmd.MarkFlagsMutuallyExclusive("data", flag)
	}

	metadataDefaultsPatchCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)
	metadataDefaultsPatchCmd.RegisterFlagCompletionFunc("patch-method", cli.CompletionPatchMethod)

	return metadataDefaultsPatchCmd
}
