// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package peer

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	metadata_service_lib "github.com/OpenCHAMI/ochami/internal/cli/metadata_service"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
)

var (
	formatPatch client.PatchMethod = client.PatchMethodRFC7386

	setList    []string
	unsetList  []string
	addList    []string
	removeList []string
)

func newCmdMetadataPeerPatch() *cobra.Command {
	// metadataPeerPatchCmd represents the "metadata peer patch" command
	var metadataPeerPatchCmd = &cobra.Command{
		Use:   "patch <uid>",
		Args:  cobra.ExactArgs(1),
		Short: "Patch an existing WireGuard peer spec",
		Long: `Patch an existing WireGuard peer spec using various patch formats.

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
  ochami metadata peer patch wireguardpeer-d614b918 --patch-method rfc6902 --data '[
    {"op":"replace","path":"/allowed_ip","value":"10.42.2.1/32"},
    {"op":"replace","path":"/description","value":"Updated peer"}
  ]'

  # Patch specific fields using JSON merge patch (RFC 7386) (simple merge)
  ochami metadata peer patch wireguardpeer-d614b918 --patch-method rfc7386 --data '{"allowed_ip":"10.42.2.1/32"}'

  # Patch specific fields using dot notation for keys (shorthand patch)
  ochami metadata peer patch wireguardpeer-d614b918 --patch-method keyval --set allowed_ip='10.42.2.1/32'

  # Patch using payload file
  ochami metadata peer patch wireguardpeer-d614b918 -d @payload.json
  ochami metadata peer patch wireguardpeer-d614b918 -d @payload.yaml -f yaml

  # Patch using stdin
  echo '<json_data>' | ochami metadata peer patch wireguardpeer-d614b918 -d @-
  echo '<json_data>' | ochami metadata peer patch wireguardpeer-d614b918
  echo '<yaml_data>' | ochami metadata peer patch wireguardpeer-d614b918 -f yaml -d @-
  echo '<yaml_data>' | ochami metadata peer patch wireguardpeer-d614b918 -f yaml`,
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

			peerPatched, err := metadataServiceClient.PatchWireGuardPeer(cli.Token, formatPatch, args[0], patchData)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to patch WireGuard peer")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Check that a modified item was returned
			if peerPatched == nil {
				log.Logger.Error().Msg("WireGuard peer patch returned no resource")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Print UIDs of modified items
			log.Logger.Info().Msgf("WireGuard peers patched: %+v", []string{peerPatched.Metadata.UID})
		},
	}

	// Create flags
	metadataPeerPatchCmd.Flags().StringArrayVar(&setList, "set", nil, "set field value using dot notation (field=value)")
	metadataPeerPatchCmd.Flags().StringArrayVar(&unsetList, "unset", nil, "unset field using dot notation")
	metadataPeerPatchCmd.Flags().StringArrayVar(&addList, "add", nil, "add value to array field (field=value)")
	metadataPeerPatchCmd.Flags().StringArrayVar(&removeList, "remove", nil, "remove value from array field (field=value)")
	metadataPeerPatchCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	metadataPeerPatchCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data for JSON patch formats (json,json-pretty,yaml)")
	metadataPeerPatchCmd.Flags().VarP(&formatPatch, "patch-method", "p", "type of patch to use (rfc6902,rfc7386,keyval)")

	for _, flag := range []string{"set", "unset", "add", "remove"} {
		metadataPeerPatchCmd.MarkFlagsMutuallyExclusive("format-input", flag)
		metadataPeerPatchCmd.MarkFlagsMutuallyExclusive("data", flag)
	}

	metadataPeerPatchCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return metadataPeerPatchCmd
}
