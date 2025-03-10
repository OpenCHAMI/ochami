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

// groupMemberAddCmd represents the smd-group-member-add command
var groupMemberAddCmd = &cobra.Command{
	Use:   "add <group_label> <component>...",
	Args:  cobra.MinimumNArgs(2),
	Short: "Add one or more components to a group",
	Long: `Add one or more components to a group.

See ochami-smd(1) for more details.`,
	Example: `  ochami smd group member add compute x3000c1s7b56n0`,
	Run: func(cmd *cobra.Command, args []string) {
		// Without a base URI, we cannot do anything
		smdBaseURI, err := getBaseURISMD(cmd)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for SMD")
			logHelpError(cmd)
			os.Exit(1)
		}

		// This endpoint requires authentication, so a token is needed
		setTokenFromEnvVar(cmd)
		checkToken(cmd)

		// Create client to make request to SMD
		smdClient, err := smd.NewClient(smdBaseURI, insecure)
		if err != nil {
			log.Logger.Error().Err(err).Msg("error creating new SMD client")
			logHelpError(cmd)
			os.Exit(1)
		}

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(smdClient.OchamiClient)

		// Send off request
		_, errs, err := smdClient.PostGroupMembers(token, args[0], args[1:]...)
		if err != nil {
			log.Logger.Error().Err(err).Msgf("failed to add group member(s) to group %s in SMD", args[0])
			logHelpError(cmd)
			os.Exit(1)
		}
		// Since smdClient.PostGroupMembers does the addition iteratively, we need to deal with
		// each error that might have occurred.
		var errorsOccurred = false
		for _, err := range errs {
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msgf("SMD group member request for group %s yielded unsuccessful HTTP response", args[0])
				} else {
					log.Logger.Error().Err(err).Msgf("failed to add group member(s) to group %s in SMD", args[0])
				}
				errorsOccurred = true
			}
		}
		if errorsOccurred {
			logHelpError(cmd)
			log.Logger.Warn().Msg("SMD group addition completed with errors")
			os.Exit(1)
		}
	},
}

func init() {
	groupMemberCmd.AddCommand(groupMemberAddCmd)
}
