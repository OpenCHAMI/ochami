// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package instance

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"
)

func newCmdMetadataInstancePatch() *cobra.Command {
	formatPatch := client.PatchMethodRFC7386
	var setList, unsetList, addList, removeList []string
	// metadataInstancePatchCmd represents the "metadata instance patch" command
	var metadataInstancePatchCmd = &cobra.Command{
		Use:   "patch <uid>",
		Args:  cobra.ExactArgs(1),
		Short: "Patch an existing instance info spec",
		Long: `Patch an existing instance info spec using various patch formats.

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
   ochami metadata instance patch instanceinfo-d614b918 --patch-method rfc6902 --data '[
     {"op":"replace","path":"/hostname","value":"nid002000.demo.cluster"},
     {"op":"replace","path":"/local_hostname","value":"nid002000"}
   ]'

  # Patch specific fields using JSON merge patch (RFC 7386) (simple merge)
  ochami metadata instance patch instanceinfo-d614b918 --patch-method rfc7386 --data '{"hostname":"nid002000.demo.cluster"}'

  # Patch specific fields using dot notation for keys (shorthand patch)
  ochami metadata instance patch instanceinfo-d614b918 --patch-method keyval --set hostname='nid002000.demo.cluster'

  # Patch using payload file
  ochami metadata instance patch instanceinfo-d614b918 -d @payload.json
  ochami metadata instance patch instanceinfo-d614b918 -d @payload.yaml -f yaml

  # Patch using stdin
  echo '<json_data>' | odami metadata instance patch instanceinfo-d614b918 -d @-
  echo '<json_data>' | odami metadata instance patch instanceinfo-d614b918
  echo '<yaml_data>' | odami metadata instance patch instanceinfo-d614b918 -d @- -f yaml
  echo '<yaml_data>' | odami metadata instance patch instanceinfo-d614b918 -f yaml`,
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

			instancePatched, err := metadataServiceClient.PatchInstanceInfo(cli.Token, formatPatch, args[0], patchData)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					return cli.Errorf(cli.CodeHTTP, "failed to patch instance info: %w", err)
				}
				return cli.Errorf(cli.CodeNetwork, "failed to patch instance info: %w", err)
			}

			// Check that a modified item was returned
			if instancePatched == nil {
				return cli.Errorf(cli.CodeGeneric, "instance info patch returned no resource")
			}

			// Print UIDs of modified items
			log.Logger.Info().Msgf("Instance infos patched: %+v", []string{instancePatched.Metadata.UID})

			return nil
		},
	}

	// Create flags
	metadataInstancePatchCmd.Flags().StringArrayVar(&setList, "set", nil, "set field value using dot notation (field=value)")
	metadataInstancePatchCmd.Flags().StringArrayVar(&unsetList, "unset", nil, "unset field using dot notation")
	metadataInstancePatchCmd.Flags().StringArrayVar(&addList, "add", nil, "add value to array field (field=value)")
	metadataInstancePatchCmd.Flags().StringArrayVar(&removeList, "remove", nil, "remove value from array field (field=value)")
	metadataInstancePatchCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	metadataInstancePatchCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data for JSON patch formats (json,json-pretty,yaml)")
	metadataInstancePatchCmd.Flags().VarP(&formatPatch, "patch-method", "p", "type of patch to use (rfc6902,rfc7386,keyval)")

	for _, flag := range []string{"set", "unset", "add", "remove"} {
		metadataInstancePatchCmd.MarkFlagsMutuallyExclusive("format-input", flag)
		metadataInstancePatchCmd.MarkFlagsMutuallyExclusive("data", flag)
	}

	metadataInstancePatchCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)
	metadataInstancePatchCmd.RegisterFlagCompletionFunc("patch-method", cli.CompletionPatchMethod)

	return metadataInstancePatchCmd
}
