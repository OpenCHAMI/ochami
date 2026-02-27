// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package bmc

import (
	"os"

	"github.com/spf13/cobra"

	boot_service_lib "github.com/OpenCHAMI/ochami/internal/cli/boot_service"
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

func newCmdBootBmcPatch() *cobra.Command {
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

Otherwise, --data can be used to pass in raw patch data, using --patch-format
to specify the patch format (see examples below).

--format-input can only be used with --data. It can be used to tell ochami to
use a different format (e.g. YAML) for the data input for either of these
methods.

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
  echo '<yaml_data>' | ochami boot bmc patch bmc-773d99bf -f yaml -d @-`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			bootServiceClient := boot_service_lib.GetClient(cmd)

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
				cli.HandlePayload(cmd, &patchData)
			}

			bmcPatched, err := bootServiceClient.PatchBMC(cli.Token, formatPatch, args[0], patchData)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to patch BMC")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			log.Logger.Debug().Msgf("BMC patched: %+v", bmcPatched)
		},
	}

	// Create flags
	bootBmcPatchCmd.Flags().StringArrayVar(&setList, "set", nil, "set field value using dot notation (field=value)")
	bootBmcPatchCmd.Flags().StringArrayVar(&unsetList, "unset", nil, "unset field using dot notation")
	bootBmcPatchCmd.Flags().StringArrayVar(&addList, "add", nil, "add value to array field (field=value)")
	bootBmcPatchCmd.Flags().StringArrayVar(&removeList, "remove", nil, "remove value from array field (field=value)")
	bootBmcPatchCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	bootBmcPatchCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data for JSON patch formats (json,json-pretty,yaml)")
	bootBmcPatchCmd.Flags().VarP(&formatPatch, "patch-method", "p", "type of patch to use (rfc6902,rfc7386,keyval)")

	for _, flag := range []string{"set", "unset", "add", "remove"} {
		bootBmcPatchCmd.MarkFlagsMutuallyExclusive("format-input", flag)
		bootBmcPatchCmd.MarkFlagsMutuallyExclusive("data", flag)
	}
	bootBmcPatchCmd.MarkFlagsOneRequired("data", "set", "unset", "add", "remove")

	bootBmcPatchCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return bootBmcPatchCmd
}
