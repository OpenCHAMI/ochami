// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"os"

	"github.com/OpenCHAMI/ochami/internal/config"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/ci"
	"github.com/spf13/cobra"
)

// cloudInitConfigDeleteCmd represents the cloud-init-config-delete command
var cloudInitConfigDeleteCmd = &cobra.Command{
	Use:     "delete <id>...",
	Args:    cobra.MinimumNArgs(1),
	Short:   "Delete one or more cloud-init configs",
	Example: `  ochami cloud-init config delete compute`,
	Run: func(cmd *cobra.Command, args []string) {
		// Without a base URI, we cannot do anything
		cloudInitBaseURI, err := getBaseURI(cmd, config.ServiceCloudInit)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for cloud-init")
			os.Exit(1)
		}

		// This endpoint requires authentication, so a token is needed
		setTokenFromEnvVar(cmd)
		checkToken(cmd)

		// Create client to make request to cloud-init
		cloudInitClient, err := ci.NewClient(cloudInitBaseURI, insecure)
		if err != nil {
			log.Logger.Error().Err(err).Msg("error creating new cloud-init client")
			os.Exit(1)
		}

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(cloudInitClient.OchamiClient)

		// Ask before attempting deletion unless --force was passed
		if !cmd.Flag("force").Changed {
			log.Logger.Debug().Msg("--force not passed, prompting user to confirm deletion")
			respDelete := loopYesNo("Really delete?")
			if !respDelete {
				log.Logger.Info().Msg("User aborted cloud-init config deletion")
				os.Exit(0)
			} else {
				log.Logger.Debug().Msg("User answered affirmatively to delete cloud-init config(s)")
			}
		}

		// Send off request
		var errs []error
		if cloudInitCmd.Flag("secure").Changed {
			_, errs, err = cloudInitClient.DeleteConfigsSecure(token, args...)
		} else {
			_, errs, err = cloudInitClient.DeleteConfigs(token, args...)
		}
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to delete cloud-init configs")
			os.Exit(1)
		}
		// Since cloudInitClient.Delete* functions do the deletion iteratively, we need to deal with
		// each error that might have occurred.
		var errorsOccurred = false
		for _, e := range errs {
			if e != nil {
				if errors.Is(e, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(e).Msg("cloud-init config request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(e).Msg("failed to delete config(s) to cloud-init")
				}
				errorsOccurred = true
			}
		}
		// Warn the user if any errors occurred during deletion iterations
		if errorsOccurred {
			log.Logger.Warn().Msg("cloud-init config deletion completed with errors")
			os.Exit(1)
		}
	},
}

func init() {
	cloudInitConfigDeleteCmd.Flags().Bool("force", false, "do not ask before attempting deletion")
	cloudInitConfigCmd.AddCommand(cloudInitConfigDeleteCmd)
}
