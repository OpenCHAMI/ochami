// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"fmt"
	"os"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/ci"
	"github.com/spf13/cobra"
)

// cloudInitDefaultsGetCmd represents the cloud-init-defaults-get command
var cloudInitDefaultsGetCmd = &cobra.Command{
	Use:     "get",
	Args:    cobra.NoArgs,
	Short:   "Get cloud-init default meta-data for a cluster",
	Example: `  ochami cloud-init defaults get`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// First and foremost, make sure config is loaded and logging
		// works.
		initConfigAndLogging(cmd, true)

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Without a base URI, we cannot do anything
		cloudInitbaseURI, err := getBaseURICloudInit(cmd)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for cloud-init")
			os.Exit(1)
		}

		// Create client to make request to cloud-init
		cloudInitClient, err := ci.NewClient(cloudInitbaseURI, insecure)
		if err != nil {
			log.Logger.Error().Err(err).Msg("error creating new cloud-init client")
			os.Exit(1)
		}

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(cloudInitClient.OchamiClient)

		// Get data
		henv, err := cloudInitClient.GetDefaults()
		if err != nil {
			log.Logger.Error().Err(err).Msgf("failed to get defaults")
			os.Exit(1)
		}

		// Print in desired format
		outFmt, err := cmd.Flags().GetString("format-output")
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get value for --format-output")
			os.Exit(1)
		}
		if outBytes, err := client.FormatBody(henv.Body, outFmt); err != nil {
			log.Logger.Error().Err(err).Msg("failed to format output")
			os.Exit(1)
		} else {
			fmt.Printf(string(outBytes))
		}
	},
}

func init() {
	cloudInitDefaultsGetCmd.Flags().StringP("format-output", "F", defaultOutputFormat, "format of output printed to standard output")
	cloudInitDefaultsCmd.AddCommand(cloudInitDefaultsGetCmd)
}
