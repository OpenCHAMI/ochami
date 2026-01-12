// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/cloud-init/pkg/cistore"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"

	cloud_init_lib "github.com/OpenCHAMI/ochami/internal/cli/cloud_init"
)

func newCmdGroupAdd() *cobra.Command {
	// groupAddCmd represents the "cloud-init group add" command
	var groupAddCmd = &cobra.Command{
		Use:   "add [-d (<data> | @<path>)] [-f <format>]",
		Args:  cobra.NoArgs,
		Short: "Add one or more new groups to cloud-init",
		Long: `Add one or more new groups to cloud-init. Data is read from
standard input. Alternatively, pass -d to pass raw payload data
or (if flag argument starts with @) a file containing the payload
data. -f can be specified to change the format of the input
payload data ('json' by default), but the rules above still apply
for the payload. If "-" is used as the input payload filename, the
data is read from standard input.

See ochami-cloud-init(1) for more details.`,
		Example: `  # Add cloud-init groups using input payload data
  ochami cloud-init group add -d '[{
    "description": "The compute group",
    "file": {
      "content": "IyMgdGVtcGxhdGU6IGppbmphCiNjbG91ZC1jb25maWcKbWVyZ2VfaG93OgotIG5hbWU6IGxpc3QKICBzZXR0aW5nczogW2FwcGVuZF0KLSBuYW1lOiBkaWN0CiAgc2V0dGluZ3M6IFtub19yZXBsYWNlLCByZWN1cnNlX2xpc3RdCnVzZXJzOgogIC0gbmFtZTogcm9vdAogICAgc3NoX2F1dGhvcml6ZWRfa2V5czoge3sgZHMubWV0YV9kYXRhLmluc3RhbmNlX2RhdGEudjEucHVibGljX2tleXMgfX0KZGlzYWJsZV9yb290OiBmYWxzZQo=",
      "encoding": "base64",
      "filename": "string"
    },
    "meta-data": {
      "foo": "bar"
    },
    "name": "compute"
  }]'

  # Add cloud-init groups using input payload file
  ochami cloud-init group add -d @payload.json
  ochami cloud-init group add -d @payload.yaml -f yaml

  # Add cloud-init groups using data from standard input
  echo '<json_data>' | ochami cloud-init group add
  echo '<json_data>' | ochami cloud-init group add -d @-
  echo '<yaml_data>' | ochami cloud-init group add -f yaml
  echo '<yaml_data>' | ochami cloud-init group add -d @- -f yaml`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			cloudInitClient := cloud_init_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// The group data we will send
			ciGroups := []cistore.GroupData{}

			// Read payload from file or stdin.
			if cmd.Flag("data").Changed {
				cli.HandlePayload(cmd, &ciGroups)
			} else {
				cli.HandlePayloadStdin(cmd, &ciGroups)
			}

			// Send data
			_, errs, err := cloudInitClient.PostGroups(ciGroups, cli.Token)
			if err != nil {
				log.Logger.Error().Err(err).Msgf("failed to add groups")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
			// Since the requests are done iteratively, we need to deal with
			// each error that might have occurred.
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(err).Msg("cloud-init group request yielded unsuccessful HTTP response")
					} else {
						log.Logger.Error().Err(err).Msg("failed to add new group to cloud-init")
					}
					errorsOccurred = true
				}
			}
			if errorsOccurred {
				log.Logger.Warn().Msg("cloud-init group addition completed with errors")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
		},
	}

	// Create flags
	groupAddCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")
	groupAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")

	groupAddCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return groupAddCmd
}
