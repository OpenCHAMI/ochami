// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"os"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client/ci"
	"github.com/spf13/cobra"
)

// cloudInitGetClient sets up the cloud-init client with the cloud-init base URI
// and certificates (if necessary) and returns it. This function is used by each
// subcommand.
func cloudInitGetClient(cmd *cobra.Command) *ci.CloudInitClient {
	// Without a base URI, we cannot do anything
	cloudInitbaseURI, err := getBaseURICloudInit(cmd)
	if err != nil {
		log.Logger.Error().Err(err).Msg("failed to get base URI for cloud-init")
		logHelpError(cmd)
		os.Exit(1)
	}

	// Create client to make request to cloud-init
	cloudInitClient, err := ci.NewClient(cloudInitbaseURI, insecure)
	if err != nil {
		log.Logger.Error().Err(err).Msg("error creating new cloud-init client")
		logHelpError(cmd)
		os.Exit(1)
	}

	// Check if a CA certificate was passed and load it into client if valid
	useCACert(cloudInitClient.OchamiClient)

	return cloudInitClient
}

// cloudInitCmd represents the cloud-init command
var cloudInitCmd = &cobra.Command{
	Use:   "cloud-init",
	Args:  cobra.NoArgs,
	Short: "Interact with the cloud-init service",
	Long: `Interact with the cloud-init service. This is a metacommand.

See ochami-cloud-init(1) for more details.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		}
	},
}

func init() {
	cloudInitCmd.PersistentFlags().String("uri", "", "absolute base URI or relative base path of cloud-init")
	rootCmd.AddCommand(cloudInitCmd)
}
