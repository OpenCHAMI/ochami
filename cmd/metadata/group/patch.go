// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	metadata_service_lib "github.com/openchami/ochami/internal/cli/metadata_service"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"
)

func newCmdMetadataGroupPatch() *cobra.Command {
	formatPatch := client.PatchMethodRFC7386
	var setList, unsetList, addList, removeList []string
	// metadataGroupPatchCmd represents the "metadata group patch" command
	var metadataGroupPatchCmd = &cobra.Command{
		Use:   "patch <uid>",
		Args:  cobra.ExactArgs(1),
		Short: "Patch an existing group spec",
		Long: `Patch an existing group spec using various patch formats.

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
   ochami metadata group patch group-d614b918 --patch-method rfc6902 --data '[
     {"op":"replace","path":"/template","value":"#cloud-config\npackages:\n  - vim\n"},
     {"op":"replace","path":"/osVersion","value":"ubuntu-24.04"}
   ]'

  # Patch specific fields using JSON merge patch (RFC 7386) (simple merge)
  ochami metadata group patch group-d614b918 --patch-method rfc7386 --data '{"osVersion":"ubuntu-24.04"}'

  # Patch specific fields using dot notation for keys (shorthand patch)
  ochami metadata group patch group-d614b918 --patch-method keyval --set osVersion='ubuntu-24.04'

  # Patch using payload file
  ochami metadata group patch group-d614b918 -d @payload.json
  ochami metadata group patch group-d614b918 -d @payload.yaml -f yaml

  # Patch using stdin
  echo '<json_data>' | ochami metadata group patch group-d614b918 -d @-
  echo '<json_data>' | ochami metadata group patch group-d614b918
  echo '<yaml_data>' | ochami metadata group patch group-d614b918 -f yaml -d @-
  echo '<yaml_data>' | ochami metadata group patch group-d614b918 -f yaml`,
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

			groupPatched, err := metadataServiceClient.PatchGroup(cmd.Context(), cli.Token, formatPatch, args[0], patchData)
			if err != nil {
				return cli.ClassifyClientError(err, "failed to patch group", "failed to patch group")

			}

			// Check that a modified item was returned
			if groupPatched == nil {
				return cli.Errorf(cli.CodeGeneric, "group patch returned no resource")
			}

			// Print UIDs of modified items
			log.Logger.Info().Msgf("Groups patched: %+v", []string{groupPatched.Metadata.UID})

			return nil
		},
	}

	// Create flags
	metadataGroupPatchCmd.Flags().StringArrayVar(&setList, "set", nil, "set field value using dot notation (field=value)")
	metadataGroupPatchCmd.Flags().StringArrayVar(&unsetList, "unset", nil, "unset field using dot notation")
	metadataGroupPatchCmd.Flags().StringArrayVar(&addList, "add", nil, "add value to array field (field=value)")
	metadataGroupPatchCmd.Flags().StringArrayVar(&removeList, "remove", nil, "remove value from array field (field=value)")
	metadataGroupPatchCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	metadataGroupPatchCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data for JSON patch formats (json,json-pretty,yaml)")
	metadataGroupPatchCmd.Flags().VarP(&formatPatch, "patch-method", "p", "type of patch to use (rfc6902,rfc7386,keyval)")

	for _, flag := range []string{"set", "unset", "add", "remove"} {
		metadataGroupPatchCmd.MarkFlagsMutuallyExclusive("format-input", flag)
		metadataGroupPatchCmd.MarkFlagsMutuallyExclusive("data", flag)
	}

	metadataGroupPatchCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)
	metadataGroupPatchCmd.RegisterFlagCompletionFunc("patch-method", cli.CompletionPatchMethod)

	return metadataGroupPatchCmd
}
