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

func newCmdGroupSet() *cobra.Command {
	// groupSetCmd represents the "cloud-init group set" command
	var groupSetCmd = &cobra.Command{
		Use:   "set [-d (<data> | @<path>)] [-f <format>]",
		Args:  cobra.NoArgs,
		Short: "Set cloud-init group data, overwriting existing data",
		Long: `Set cloud-init group data, overwriting existing data. Data is read from
standard input. Alternatively, pass -d to pass raw payload data
or (if flag argument starts with @) a file containing the payload
data. -f can be specified to change the format of the input
payload data ('json' by default), but the rules above still apply
for the payload. If "-" is used as the input payload filename, the
data is read from standard input.

See ochami-cloud-init(1) for more details.`,
		Example: `  # Set cloud-init group data using input payload data
  ochami cloud-init group set -d '[{
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

  # Set cloud-init group data using input payload file
  ochami cloud-init group set -d @payload.json
  ochami cloud-init group set -d @payload.yaml -f yaml

  # Set cloud-init group data using data from standard input
  echo '<json_data>' | ochami cloud-init group set
  echo '<json_data>' | ochami cloud-init group set -d @-
  echo '<yaml_data>' | ochami cloud-init group set -f yaml
  echo '<yaml_data>' | ochami cloud-init group set -d @- -f yaml`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			cloudInitClient := cloud_init_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// The list of group data we will send
			ciGroups := []cistore.GroupData{}

			// Read payload from file or stdin.
			if cmd.Flag("data").Changed {
				cli.HandlePayload(cmd, &ciGroups)
			} else {
				cli.HandlePayloadStdin(cmd, &ciGroups)
			}

			// Send data
			_, errs, err := cloudInitClient.PutGroups(ciGroups, cli.Token)
			if err != nil {
				log.Logger.Error().Err(err).Msgf("failed to set group data")
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
						log.Logger.Error().Err(err).Msg("failed to set group data in cloud-init")
					}
					errorsOccurred = true
				}
			}
			if errorsOccurred {
				log.Logger.Warn().Msg("cloud-init group data setting completed with errors")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
		},
	}

	// Create flags
	groupSetCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")
	groupSetCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")

	groupSetCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return groupSetCmd
}
