// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"os"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/smd"
	"github.com/spf13/cobra"
)

// groupMemberDeleteCmd represents the smd-group-member-delete command
var groupMemberDeleteCmd = &cobra.Command{
	Use:   "delete <group_label> <component>...",
	Args:  cobra.MinimumNArgs(2),
	Short: "Delete one or more members from a group",
	Run: func(cmd *cobra.Command, args []string) {
		// Without a base URI, we cannot do anything
		smdBaseURI, err := getBaseURISMD(cmd)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for SMD")
			os.Exit(1)
		}

		// This endpoint requires authentication, so a token is needed
		setTokenFromEnvVar(cmd)
		checkToken(cmd)

		// Create client to make request to SMD
		smdClient, err := smd.NewClient(smdBaseURI, insecure)
		if err != nil {
			log.Logger.Error().Err(err).Msg("error creating new SMD client")
			os.Exit(1)
		}

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(smdClient.OchamiClient)

		// Ask before attempting deletion unless --force was passed
		if !cmd.Flag("force").Changed {
			log.Logger.Debug().Msg("--force not passed, prompting user to confirm deletion")
			respDelete := loopYesNo("Really delete?")
			if !respDelete {
				log.Logger.Info().Msg("User aborted group deletion")
				os.Exit(0)
			} else {
				log.Logger.Debug().Msg("User answered affirmatively to delete groups members")
			}
		}

		// Perform deletion from arguments
		_, errs, err := smdClient.DeleteGroupMembers(token, args[0], args[1:]...)
		if err != nil {
			log.Logger.Error().Err(err).Msgf("failed to delete members from group %s in SMD", args[0])
			os.Exit(1)
		}
		// Since smdClient.DeleteGroupMembers does the deletion iteratively, we need to deal with
		// each error that might have occurred.
		var errorsOccurred = false
		for _, e := range errs {
			if err != nil {
				if errors.Is(e, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(e).Msg("SMD group member deletion yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(e).Msg("failed to delete group member(s)")
				}
				errorsOccurred = true
			}
		}
		// Warn the user if any errors occurred during deletion iterations
		if errorsOccurred {
			log.Logger.Warn().Msg("SMD group member deletion completed with errors")
			os.Exit(1)
		}
	},
}

func init() {
	groupMemberDeleteCmd.Flags().Bool("force", false, "do not ask before attempting deletion")
	groupMemberCmd.AddCommand(groupMemberDeleteCmd)
}
