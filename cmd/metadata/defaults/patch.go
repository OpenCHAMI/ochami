// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package defaults

import (
	"os"

	"github.com/spf13/cobra"

	metadata_service_lib "github.com/OpenCHAMI/ochami/internal/cli/metadata_service"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"

	"github.com/OpenCHAMI/ochami/internal/cli"
)

var (
	formatPatch client.PatchMethod = client.PatchMethodRFC7386

	setList    []string
	unsetList  []string
	addList    []string
	removeList []string
)

func newCmdMetadataDefaultsPatch() *cobra.Command {
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
    {"op":"replace","path":"short_name","value":"de"}
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
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			metadataServiceClient := metadata_service_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			var patchData map[string]interface{}
			if cmd.Flag("set").Changed || cmd.Flag("unset").Changed || cmd.Flag("add").Changed || cmd.Flag("remove").Changed {
				if cmd.Flag("patch-format").Changed && formatPatch != client.PatchMethodKeyVal {
					log.Logger.Warn().Msg("overriding --patch-format since --set/--unset/--add/--remove was passed")
				}

				pd, err := client.NewKeyValPatch(setList, unsetList, addList, removeList)
				if err != nil {
					log.Logger.Error().Err(err).Msg("error creating key-value patch data")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}
				patchData = pd
			} else {
				if cmd.Flag("data").Changed {
					cli.HandlePayload(cmd, &patchData)
				} else {
					cli.HandlePayloadStdin(cmd, &patchData)
				}
			}

			defaultsPatched, err := metadataServiceClient.PatchDefaults(cli.Token, formatPatch, args[0], patchData)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to patch cluster defaults")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			log.Logger.Debug().Msgf("cluster defaults patched: %+v", defaultsPatched)
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

	return metadataDefaultsPatchCmd
}
