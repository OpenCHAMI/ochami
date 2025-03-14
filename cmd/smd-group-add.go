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

// groupAddCmd represents the smd-group-add command
var groupAddCmd = &cobra.Command{
	Use:   "add -f <payload_file> | <group_label>",
	Args:  cobra.MaximumNArgs(1),
	Short: "Add new group",
	Long: `Add new group. A group name is required unless -f is passed to read the payload file.
Specifying -f also is mutually exclusive with the other flags of this commands
and its arguments. If - is used as the argument to -f, the data is read from
standard input.

This command sends a POST to SMD. An access token is required.`,
	Example: `  ochami smd group add computes
  ochami smd group add -d "Compute group" computes
  ochami smd group add -d "Compute group" --tag tag1,tag2 --m x3000c1s7b0n1,x3000c1s7b1n1 computes
  ochami smd group add \
    --description "ARM64 group" \
    --tag arm,64-bit \
    --member x3000c1s7b0n1,x3000c1s7b1n1 \
    --exclusive-group amd64 \
    arm64
  ochami smd group add -f payload.json
  ochami smd group add -f payload.yaml --payload-format yaml
  echo '<json_data>' | ochami smd group add -f -
  echo '<yaml_data>' | ochami smd group add -f - --payload-format yaml`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Check that all required args are passed
		if len(args) == 0 && !cmd.Flag("payload").Changed {
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

		var groups []smd.Group
		if cmd.Flag("payload").Changed {
			// Use payload file if passed
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
			if cmd.Flag("exclusive-group").Changed {
				if group.ExclusiveGroup, err = cmd.Flags().GetString("exclusive-group"); err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch exclusive group name")
					os.Exit(1)
				}
			}
			if cmd.Flag("member").Changed {
				if group.Members.IDs, err = cmd.Flags().GetStringSlice("member"); err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch members")
					os.Exit(1)
				}
			}
			groups = append(groups, group)
		}

		// Send off request
		_, errs, err := smdClient.PostGroups(groups, token)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to add group to SMD")
			os.Exit(1)
		}
		// Since smdClient.PostGroups does the addition iteratively, we need to deal with
		// each error that might have occurred.
		var errorsOccurred = false
		for _, err := range errs {
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("SMD group request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to add group(s) to SMD")
				}
				errorsOccurred = true
			}
		}
		if errorsOccurred {
			log.Logger.Warn().Msg("SMD group addition completed with errors")
			os.Exit(1)
		}
	},
}

func init() {
	groupAddCmd.Flags().StringP("description", "d", "", "brief description of group")
	groupAddCmd.Flags().StringSlice("tag", []string{}, "one or more tags for group")
	groupAddCmd.Flags().StringP("exclusive-group", "e", "", "name of group that cannot share members with this one")
	groupAddCmd.Flags().StringSliceP("member", "m", []string{}, "one or more component IDs to add to the new group")
	groupAddCmd.Flags().StringP("payload", "f", "", "file containing the request payload; JSON format unless --payload-format specified")
	groupAddCmd.Flags().StringP("payload-format", "F", defaultPayloadFormat, "format of payload file (yaml,json) passed with --payload")

	groupAddCmd.MarkFlagsMutuallyExclusive("description", "payload")
	groupAddCmd.MarkFlagsMutuallyExclusive("tag", "payload")
	groupAddCmd.MarkFlagsMutuallyExclusive("exclusive-group", "payload")
	groupAddCmd.MarkFlagsMutuallyExclusive("member", "payload")

	groupCmd.AddCommand(groupAddCmd)
}
