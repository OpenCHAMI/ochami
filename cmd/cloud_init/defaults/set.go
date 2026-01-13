// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package defaults

import (
	"os"

	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"

	cloud_init_lib "github.com/OpenCHAMI/ochami/internal/cli/cloud_init"
)

func newCmdDefaultsSet() *cobra.Command {
	// defaultsSetCmd represents the "cloud-init defaults set" command
	var defaultsSetCmd = &cobra.Command{
		Use:   "set [-d (<data> | @<path>)] [-f <format>]",
		Args:  cobra.NoArgs,
		Short: "Set default meta-data for cluster in cloud-init",
		Long: `Set default meta-data for cluster in cloud-init. Pass -d to pass raw payload
data or (if flag argument starts with @) a file containing the payload
data. -f can be specified to change the format of the input payload
data ('json' by default), but the rules above still apply for the
payload. If "-" is used as the input payload filename, the data is read
from standard input. If -d is not passed, the data is read from stdin.

See ochami-cloud-init(1) for more details.`,
		Example: `  # Set cloud-init defaults using input payload data
  ochami cloud-init defaults set -d '{
    "availability-zone": "string",
    "base-url": "http://demo.openchami.cluster:8081/cloud-init",
    "boot-subnet": "string",
    "cloud_provider": "string",
    "cluster-name": "demo",
    "nid-length": 3,
    "public-keys": [
      "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMLtQNuzGcMDatF+YVMMkuxbX2c5v2OxWftBhEVfFb+U user1@demo-head",
      "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIB4vVRvkzmGE5PyWX2fuzJEgEfET4PRLHXCnD1uFZ8ZL user2@demo-head"
    ],
    "region": "string",
    "short-name": "nid",
    "wg-subnet": "string"
  }'

  # Set cloud-init defaults using input payload file
  ochami cloud-init defaults set -d @payload.json
  ochami cloud-init defaults set -d @payload.yaml -f yaml

  # Set cloud-init defaults using data from standard input
  echo '<json_data>' | ochami cloud-init defaults set
  echo '<json_data>' | ochami cloud-init defaults set -d @-
  echo '<yaml_data>' | ochami cloud-init defaults set -f yaml
  echo '<yaml_data>' | ochami cloud-init defaults set -d @- -f yaml`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			cloudInitClient := cloud_init_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// The ClusterDefaults data we will send
			ciDflts := cistore.ClusterDefaults{}

			// Read payload from file or stdin.
			if cmd.Flag("data").Changed {
				cli.HandlePayload(cmd, &ciDflts)
			} else {
				cli.HandlePayloadStdin(cmd, &ciDflts)
			}

			// Send data
			if _, err := cloudInitClient.PostDefaults(ciDflts, cli.Token); err != nil {
				log.Logger.Error().Err(err).Msgf("failed to set defaults")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
		},
	}

	// Create flags
	defaultsSetCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")
	defaultsSetCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")

	defaultsSetCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return defaultsSetCmd
}
