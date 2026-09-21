// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package bmc

import (
	"github.com/spf13/cobra"

	boot_service_lib "github.com/openchami/ochami/internal/cli/boot_service"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"

	"github.com/openchami/ochami/internal/cli"
)

func newCmdBootBmcPatch() *cobra.Command {
	formatPatch := client.PatchMethodRFC7386
	var setList, unsetList, addList, removeList []string

	// bootBmcPatchCmd represents the "boot bmc patch" command
	var bootBmcPatchCmd = &cobra.Command{
		Use:   "patch <uid>",
		Args:  cobra.ExactArgs(1),
		Short: "Patch an existing BMC spec",
		Long: `Patch an existing BMC spec using various patch formats.

IMPORTANT: Only the spec portion of the resource can be patched.
Metadata (name, labels, annotations) and status are managed by the API.
Attempts to patch metadata or status fields will be ignored.

If --set/--unset/--add/--remove are specified or --patch-method is 'keyval',
the manual, key-value patch method using dot notation (e.g. key.subkey=value)
is used.

Otherwise, stdin and/or --data can be used to pass in raw patch data, using
--patch-method to specify the patch format (see examples below).

--format-input can only be used with stdin/--data. It can be used to tell
ochami to use a different format (e.g. YAML) for the data input for either
of these methods.

See ochami-boot(1) for more details.`,
		Example: `  # Patch using JSON patch (RFC 6902)
  ochami boot bmc patch bmc-773d99bf --patch-method rfc6902 --data '[
    {"op":"replace","path":"/description","value":"New description"},
    {"op":"replace","path":"/interface/ip","value":"172.16.0.253"}
  ]'

  # Patch specific fields using JSON merge patch (RFC 7386) (simple merge)
  ochami boot bmc patch bmc-773d99bf --patch-method rfc7386 --data '{"description":"New description"}'

  # Patch specific fields using dot notation for keys (shorthand patch)
  ochami boot bmc patch bmc-773d99bf --patch-method keyval --set description='New Description' --set interface.ip=172.16.0.253

  # Patch using payload file
  ochami boot bmc patch bmc-773d99bf -d @payload.json
  ochami boot bmc patch bmc-773d99bf -f yaml -d @payload.yaml

  # Patch using stdin
  echo '<json_data>' | ochami boot bmc patch bmc-773d99bf -d @-
  echo '<json_data>' | ochami boot bmc patch bmc-773d99bf
  echo '<yaml_data>' | ochami boot bmc patch bmc-773d99bf -d @- -f yaml
  echo '<yaml_data>' | ochami boot bmc patch bmc-773d99bf -f yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			bootServiceClient, err := boot_service_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			var patchData interface{}
			patchMethod := formatPatch
			if cmd.Flag("set").Changed || cmd.Flag("unset").Changed || cmd.Flag("add").Changed || cmd.Flag("remove").Changed {
				oldFormatPatch := patchMethod
				newPatchMethod, pd, err := client.NewKeyValPatchData(setList, unsetList, addList, removeList)
				if err != nil {
					return cli.Errorf(cli.CodeUsage, "error creating key-value patch data: %w", err)
				}
				patchMethod = newPatchMethod
				if cmd.Flag("patch-method").Changed && oldFormatPatch != patchMethod {
					log.Logger.Warn().Msg("overriding --patch-method since --set/--unset/--add/--remove was passed")
				}
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

			bmcPatched, err := bootServiceClient.PatchBMC(cmd.Context(), cli.Token, patchMethod, args[0], patchData)
			if err != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to patch BMC: %w", err)
			}

			log.Logger.Debug().Msgf("BMC patched: %+v", bmcPatched)

			return nil
		},
	}

	// Create flags
	bootBmcPatchCmd.Flags().StringArrayVar(&setList, "set", nil, "set field value using dot notation (field=value)")
	bootBmcPatchCmd.Flags().StringArrayVar(&unsetList, "unset", nil, "unset field using dot notation")
	bootBmcPatchCmd.Flags().StringArrayVar(&addList, "add", nil, "add value to array field (field=value)")
	bootBmcPatchCmd.Flags().StringArrayVar(&removeList, "remove", nil, "remove value from array field by index (field=index)")
	bootBmcPatchCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	bootBmcPatchCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data for JSON patch formats (json,json-pretty,yaml)")
	bootBmcPatchCmd.Flags().VarP(&formatPatch, "patch-method", "p", "type of patch to use (rfc6902,rfc7386,keyval)")

	for _, flag := range []string{"set", "unset", "add", "remove"} {
		bootBmcPatchCmd.MarkFlagsMutuallyExclusive("format-input", flag)
		bootBmcPatchCmd.MarkFlagsMutuallyExclusive("data", flag)
	}

	bootBmcPatchCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)
	bootBmcPatchCmd.RegisterFlagCompletionFunc("patch-method", cli.CompletionPatchMethod)

	return bootBmcPatchCmd
}
