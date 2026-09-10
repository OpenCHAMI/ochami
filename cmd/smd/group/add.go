// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/smd"

	smd_lib "github.com/openchami/ochami/internal/cli/smd"
)

// groupAddOptions holds the flag values for the smd group add command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type groupAddOptions struct {
	Description    string
	Tags           []string
	ExclusiveGroup string
	Members        []string
}

// runCoreGroupAdd contains the core logic for the smd group add command.
// It takes the parsed options and performs the actual work of adding groups.
func runCoreGroupAdd(cmd *cobra.Command, opts *groupAddOptions, args []string, smdClient *smd.SMDClient) error {
	// Handle token for this command
	if err := cli.HandleToken(cmd); err != nil {
		return err
	}

	// Check if a CA certificate was passed and load it into client if valid
	if err := cli.UseCACert(smdClient.OchamiClient); err != nil {
		return err
	}

	var groups []smd.Group
	if cmd.Flag("data").Changed {
		// Use payload file if passed
		if err := cli.HandlePayload(cmd, &groups); err != nil {
			return err
		}
	} else {
		// ...otherwise use CLI options/args
		group := smd.Group{Label: args[0]}
		group.Description = opts.Description
		group.Tags = opts.Tags
		group.ExclusiveGroup = opts.ExclusiveGroup
		group.Members.IDs = opts.Members
		groups = append(groups, group)
	}

	// Send off request
	results := smdClient.PostGroups(cmd.Context(), groups, cli.Token)
	// Since smdClient.PostGroups does the addition iteratively, we need to deal with
	// each error that might have occurred.
	var errorsOccurred = false
	for _, e := range results.Errors() {
		if e != nil {
			if errors.Is(e, client.UnsuccessfulHTTPError) {
				log.Logger.Error().Err(e).Msg("SMD group request yielded unsuccessful HTTP response")
			} else {
				log.Logger.Error().Err(e).Msg("failed to add group(s) to SMD")
			}
			errorsOccurred = true
		}
	}
	if errorsOccurred {
		return cli.Errorf(cli.CodeHTTP, "SMD group addition completed with errors")
	}

	return nil
}

func newCmdGroupAdd() *cobra.Command {
	// groupAddCmd represents the "smd group add" command
	var groupAddCmd = &cobra.Command{
		Use:   "add (-d (<payload_data> | @<payload_file>)) | <group_label>",
		Args:  cobra.MaximumNArgs(1),
		Short: "Add new group",
		Long: `Add new group. A group name is required. Alternatively,
pass -d to pass raw payload data or (if flag argument
starts with @) a file containing the payload data. -f
can be specified to change the format of the input payload
data ('json' by default), but the rules above still
apply for the payload. If "-" is used as the input payload
filename, the data is read from standard input.

This command sends a POST to SMD. An access token is required.

See ochami-smd(1) for more details.`,
		Example: `  # Add group using CLI flags
  ochami smd group add computes
  ochami smd group add -D "Compute group" computes
  ochami smd group add -D "Compute group" --tag tag1,tag2 --m x3000c1s7b0n1,x3000c1s7b1n1 computes
  ochami smd group add \
    --description "ARM64 group" \
    --tag arm,64-bit \
    --member x3000c1s7b0n1,x3000c1s7b1n1 \
    --exclusive-group amd64 \
    arm64

  # Add groups using input paylad data
  ochami smd group add -d '{[
    {
      "label": "computes",
      "description": "Compute group",
      "tags": ["tag1","tag2"],
      "members": {
        "ids": [
	  "x3000c1s7b0n1",
	  "x3000c1s7b1n1"
	],
      },
    }
  ]}'

  # Add groups using input payload file
  ochami smd group add -d @payload.json
  ochami smd group add -d @payload.yaml -f yaml

  # Add groups using data from standard input
  echo '<json_data>' | ochami smd group add -d @-
  echo '<yaml_data>' | ochami smd group add -d @- -f yaml`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Check that all required args are passed
			if !cmd.Flag("data").Changed {
				if len(args) != 1 {
					return cli.Errorf(cli.CodeUsage, "expected -d or 1 argument (group label), got %d", len(args))
				}
			} else {
				if len(args) > 0 {
					log.Logger.Warn().Msgf("raw data passed, ignoring CLI configuration for passed group %v", args)
				}
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			smdClient, err := smd_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Extract options from flags
			// Since flags are registered with the correct types on this command,
			// these Get* calls cannot fail, so we ignore errors with explicit comments
			opts := &groupAddOptions{}
			if cmd.Flag("description").Changed {
				opts.Description, _ = cmd.Flags().GetString("description") // Flag registered with matching type, error impossible
			}
			if cmd.Flag("tag").Changed {
				opts.Tags, _ = cmd.Flags().GetStringSlice("tag") // Flag registered with matching type, error impossible
			}
			if cmd.Flag("exclusive-group").Changed {
				opts.ExclusiveGroup, _ = cmd.Flags().GetString("exclusive-group") // Flag registered with matching type, error impossible
			}
			if cmd.Flag("member").Changed {
				opts.Members, _ = cmd.Flags().GetStringSlice("member") // Flag registered with matching type, error impossible
			}

			return runCoreGroupAdd(cmd, opts, args, smdClient)
		},
	}

	// Create flags
	groupAddCmd.Flags().StringP("description", "D", "", "brief description of group")
	groupAddCmd.Flags().StringSlice("tag", []string{}, "one or more tags for group")
	groupAddCmd.Flags().StringP("exclusive-group", "e", "", "name of group that cannot share members with this one")
	groupAddCmd.Flags().StringSliceP("member", "m", []string{}, "one or more component IDs to add to the new group")
	groupAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	groupAddCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	groupAddCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)
	groupAddCmd.MarkFlagsMutuallyExclusive("description", "data")
	groupAddCmd.MarkFlagsMutuallyExclusive("tag", "data")
	groupAddCmd.MarkFlagsMutuallyExclusive("exclusive-group", "data")
	groupAddCmd.MarkFlagsMutuallyExclusive("member", "data")

	return groupAddCmd
}
