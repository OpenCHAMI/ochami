// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/ci"
	"github.com/spf13/cobra"
)

// cloudInitGroupGetCmd represents the cloud-init-defaults-get command
var cloudInitGroupGetCmd = &cobra.Command{
	Use:     "get [id...]",
	Short:   "Get group data for all groups or for a list of group IDs",
	Example: `  ochami cloud-init group get
  ochami cloud-init group get compute`,
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
		if len(args) == 0 {
			// No args passed, get all group data at once
			henvs, errs, err := cloudInitClient.GetGroups()
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to get all groups from cloud-init")
				os.Exit(1)
			}
			if errs[0] != nil {
				if errors.Is(errs[0], client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("cloud-init group request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to cloud-init groups")
				}
				os.Exit(1)
			}

			// Group data is formatted as a map keyed on the name,
			// which is a bit awkward since the name appears twice
			// and is hard to iterate through.
			//
			// Convert group map into group slice.
			var groupSlice []cistore.GroupData
			var groupMap   map[string]cistore.GroupData
			if err := json.Unmarshal(henvs[0].Body, &groupMap); err != nil {
				log.Logger.Error().Err(err).Msg("failed to unmarshal all groups")
				os.Exit(1)
			}
			for _, group := range groupMap {
				groupSlice = append(groupSlice, group)
			}
			// Marshal slice back to JSON to be printed.
			groupSliceBytes, err := json.Marshal(groupSlice)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to marshal group list into JSON")
				os.Exit(1)
			}

			// Print in desired format
			outFmt, err := cmd.Flags().GetString("format-output")
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to get value for --format-output")
				os.Exit(1)
			}
			if outBytes, err := client.FormatBody(groupSliceBytes, outFmt); err != nil {
				log.Logger.Error().Err(err).Msg("failed to format output")
				os.Exit(1)
			} else {
				fmt.Printf(string(outBytes))
			}
		} else {
			// One or more arguments (group IDs) provided, get data
			// for just those groups.
			henvs, errs, err := cloudInitClient.GetGroups(args...)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to get cloud-init groups")
				os.Exit(1)
			}
			// Since the requests are done iteratively, we need to
			// deal with each error that might have occurred.
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(err).Msg("cloud-init group request yielded unsuccessful HTTP response")
					} else {
						log.Logger.Error().Err(err).Msg("failed to get cloud-init groups")
					}
					errorsOccurred = true
				}
			}
			if errorsOccurred {
				log.Logger.Warn().Msg("cloud-init group retrieval completed with errors")
				os.Exit(1)
			}

			// Collect group data into JSON array
			var ciGroups []cistore.GroupData
			for _, henv := range henvs {
				var ciGroup cistore.GroupData
				if err := json.Unmarshal(henv.Body, &ciGroup); err != nil {
					log.Logger.Error().Err(err).Msg("failed to unmarshal HTTP body into group")
				} else {
					ciGroups = append(ciGroups, ciGroup)
				}
			}
			jsonGroups, err := json.Marshal(ciGroups)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to marshal group list")
				os.Exit(1)
			}

			// Print in desired format
			outFmt, err := cmd.Flags().GetString("format-output")
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to get value for --format-output")
				os.Exit(1)
			}
			if outBytes, err := client.FormatBody(jsonGroups, outFmt); err != nil {
				log.Logger.Error().Err(err).Msg("failed to format output")
				os.Exit(1)
			} else {
				fmt.Printf(string(outBytes))
			}
		}
	},
}

func init() {
	cloudInitGroupGetCmd.Flags().StringP("format-output", "F", defaultOutputFormat, "format of output printed to standard output")
	cloudInitGroupCmd.AddCommand(cloudInitGroupGetCmd)
}
