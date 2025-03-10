// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/ci"
	"github.com/spf13/cobra"
)

// cloudInitGroupDeleteCmd represents the "cloud-init group delete" command
var cloudInitGroupDeleteCmd = &cobra.Command{
	Use:   "delete (-d (<data> | @<path>)) | <group>...",
	Short: "Delete one or more cloud-init groups",
	Long: `Delete one or more cloud-init groups. Either one or more group
names must be specified, or raw payload must be specified
with -d. If the argument to -d begins with @, the argument
is interpreted as a file path to read the payload data from.
If the path is -, the data is read from standard input.
-f can be specified to change the format of the input
payload data ('json' by default).

See ochami-cloud-init(1) for more details.`,
	Example: `  # Delete cloud-init groups using CLI arguments
  ochami cloud-init group delete compute my-group

  # Delete cloud-init groups using input payload data
  ochami cloud-init group delete -d '[{"name":"compute"},{"name":"my-group"}]'

  # Delete cloud-init groups using input payload file
  ochami cloud-init group delete -d @payload.json
  ochami cloud-init group delete -d @payload.yaml -f yaml

  # Delete cloud-init groups using data from standard input
  echo '<json_data>' | ochami cloud-init group delete
  echo '<json_data>' | ochami cloud-init group delete -d @-
  echo '<yaml_data>' | ochami cloud-init group delete -f yaml
  echo '<yaml_data>' | ochami cloud-init group delete -d @- -f yaml`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 && !cmd.Flag("data").Changed {
			return fmt.Errorf("either -d or at least one group name is required")
		} else if len(args) > 0 && cmd.Flag("data").Changed {
			return fmt.Errorf("either -d or at least one group name is required, but not both")
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
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

		// The ClusterDefaults data we will send
		ciGroups := []cistore.GroupData{}

		// Read payload from file or stdin.
		var groupsToDel []string
		if cmd.Flag("data").Changed {
			handlePayload(cmd, &ciGroups)
			for _, group := range ciGroups {
				groupsToDel = append(groupsToDel, group.Name)
			}
		} else {
			groupsToDel = args
		}

		// Ask before attempting deletion unless --force was passed
		if !cmd.Flag("force").Changed {
			log.Logger.Debug().Msg("--force not passed, prompting user to confirm deletion")
			respDelete := loopYesNo("Really delete?")
			if !respDelete {
				log.Logger.Info().Msg("User aborted cloud-init group deletion")
				os.Exit(0)
			} else {
				log.Logger.Debug().Msg("User answered affirmatively to delete cloud-init groups")
			}
		}

		// Send data
		_, errs, err := cloudInitClient.DeleteGroups(token, groupsToDel...)
		if err != nil {
			log.Logger.Error().Err(err).Msgf("failed to delete groups")
			logHelpError(cmd)
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
					log.Logger.Error().Err(err).Msg("failed to delete groups in cloud-init")
				}
				errorsOccurred = true
			}
		}
		if errorsOccurred {
			log.Logger.Warn().Msg("cloud-init group deletion completed with errors")
			logHelpError(cmd)
			os.Exit(1)
		}
	},
}

func init() {
	cloudInitGroupDeleteCmd.Flags().Bool("force", false, "do not ask before attempting deletion")
	cloudInitGroupDeleteCmd.Flags().VarP(&formatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")
	cloudInitGroupDeleteCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")

	cloudInitGroupDeleteCmd.RegisterFlagCompletionFunc("format-input", completionFormatData)

	cloudInitGroupCmd.AddCommand(cloudInitGroupDeleteCmd)
}
