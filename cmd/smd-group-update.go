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

// groupUpdateCmd represents the smd-group-update command
var groupUpdateCmd = &cobra.Command{
	Use:   "update -f <payload_file> | ([--description <description>] [--tag <tag>]... <group_label>)",
	Args:  cobra.MaximumNArgs(1),
	Short: "Update the description and/or tags of a group",
	Long: `Update the description and/or tags of a group. At least one of --description
or --tag must be specified. Alternatively, pass -f to pass a file
(optionally specifying --payload-format, JSON by default), but the
rules above still apply for the payload. If - is used as the
argument to -f, the data is read from standard input.

This command sends a PATCH to SMD. An access token is required.`,
	Example: `  ochami smd group update --description "New description for compute" compute
  ochami smd group update --tag existing_tag --tag new_tag compute
  ochami smd group update --tag existing_tag,new_tag compute
  ochami smd group update --tag existing_tag,new_tag -d "New description for compute" compute
  ochami smd group update -f payload.json
  ochami smd group update -f payload.yaml --payload-format yaml
  echo '<json_data>' | ochami smd group update -f -
  echo '<yaml_data>' | ochami smd group update -f - --payload-format yaml`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// cmd.LocalFlags().NFlag() doesn't seem to work, so we check every flag
		if len(args) == 0 && !cmd.Flag("description").Changed && !cmd.Flag("tag").Changed {
			printUsageHandleError(cmd)
			os.Exit(0)
		}

		return nil
	},
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

		// The group list we will send
		var groups []smd.Group

		// Read payload from file first, allowing overwrites from flags
		if cmd.Flag("payload").Changed {
			handlePayload(cmd, &groups)
		} else {
			// ...otherwise use CLI options/args
			group := smd.Group{Label: args[0]}
			if cmd.Flag("description").Changed {
				if group.Description, err = cmd.Flags().GetString("description"); err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch description")
					os.Exit(1)
				}
			}
			if cmd.Flag("tag").Changed {
				if group.Tags, err = cmd.Flags().GetStringSlice("tag"); err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch tags")
					os.Exit(1)
				}
			}
			groups = append(groups, group)
		}

		// Send 'em off
		_, errs, err := smdClient.PatchGroups(groups, token)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to patch group in SMD")
			os.Exit(1)
		}
		// Since smdClient.PatchGroups does the edition iteratively, we need to deal with
		// each error that might have occurred.
		var errorsOccurred = false
		for _, err := range errs {
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("SMD group request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to update group(s) to SMD")
				}
				errorsOccurred = true
			}
		}
		if errorsOccurred {
			log.Logger.Warn().Msg("SMD group update completed with errors")
			os.Exit(1)
		}
	},
}

func init() {
	groupUpdateCmd.Flags().StringP("description", "d", "", "short description to update group with")
	groupUpdateCmd.Flags().StringSlice("tag", []string{}, "one or more tags to set for group")
	groupUpdateCmd.Flags().StringP("payload", "f", "", "file containing the request payload; JSON format unless --payload-format specified")
	groupUpdateCmd.Flags().StringP("payload-format", "F", defaultPayloadFormat, "format of payload file (yaml,json) passed with --payload")

	groupUpdateCmd.MarkFlagsOneRequired("description", "tag", "payload")

	groupCmd.AddCommand(groupUpdateCmd)
}
