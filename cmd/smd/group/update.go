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

// groupUpdateOptions holds the flag values for the smd group update command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type groupUpdateOptions struct {
	Description string
	Tags        []string
}

// runCoreGroupUpdateWithRuntime contains the core logic for the smd group update command.
// It takes the parsed options and performs the actual work of updating groups.
// Runtime-aware version.
func runCoreGroupUpdateWithRuntime(cmd *cobra.Command, opts *groupUpdateOptions, args []string, smdClient *smd.SMDClient, rt *cli.Runtime) error {
	// Handle token for this command
	if err := rt.HandleToken(cmd); err != nil {
		return err
	}

	// The group list we will send
	var groups []smd.Group

	// Read payload from file first, allowing overwrites from flags
	if cmd.Flag("data").Changed {
		if err := rt.HandlePayload(cmd, &groups); err != nil {
			return err
		}
	} else {
		// ...otherwise use CLI options/args
		group := smd.Group{Label: args[0]}
		group.Description = opts.Description
		group.Tags = opts.Tags
		groups = append(groups, group)
	}

	// Send 'em off
	results := smdClient.PatchGroups(cmd.Context(), groups, rt.Token)
	// Since smdClient.PatchGroups does the edition iteratively, we need to deal with
	// each error that might have occurred.
	var errorsOccurred = false
	for i, result := range results {
		if result.Err != nil {
			if errors.Is(result.Err, client.UnsuccessfulHTTPError) {
				log.Logger.Error().Err(result.Err).
					Str("group", groups[i].Label).
					Msg("SMD group update request yielded unsuccessful HTTP response")
				log.Logger.Info().Msg("  - Group may not exist")
				log.Logger.Info().Msg("  - Invalid field values")
			} else {
				log.Logger.Error().Err(result.Err).
					Str("group", groups[i].Label).
					Msg("failed to update group in SMD")
			}
			errorsOccurred = true
		}
	}
	if errorsOccurred {
		return cli.Errorf(cli.CodeHTTP, "SMD group update completed with errors")
	}

	// Success, log confirmation
	log.Logger.Info().
		Int("group_count", len(groups)).
		Msg("Successfully updated group(s)")

	return nil
}

// runCoreGroupUpdate contains the core logic for the smd group update command.
// It takes the parsed options and performs the actual work of updating groups.
// Legacy version using global state.
func runCoreGroupUpdate(cmd *cobra.Command, opts *groupUpdateOptions, args []string, smdClient *smd.SMDClient) error {
	// Handle token for this command
	if err := cli.HandleToken(cmd); err != nil {
		return err
	}

	// The group list we will send
	var groups []smd.Group

	// Read payload from file first, allowing overwrites from flags
	if cmd.Flag("data").Changed {
		if err := cli.HandlePayload(cmd, &groups); err != nil {
			return err
		}
	} else {
		// ...otherwise use CLI options/args
		group := smd.Group{Label: args[0]}
		group.Description = opts.Description
		group.Tags = opts.Tags
		groups = append(groups, group)
	}

	// Send 'em off
	results := smdClient.PatchGroups(cmd.Context(), groups, cli.Token)
	// Since smdClient.PatchGroups does the edition iteratively, we need to deal with
	// each error that might have occurred.
	var errorsOccurred = false
	for i, result := range results {
		if result.Err != nil {
			if errors.Is(result.Err, client.UnsuccessfulHTTPError) {
				log.Logger.Error().Err(result.Err).
					Str("group", groups[i].Label).
					Msg("SMD group update request yielded unsuccessful HTTP response")
				log.Logger.Info().Msg("  - Group may not exist")
				log.Logger.Info().Msg("  - Invalid field values")
			} else {
				log.Logger.Error().Err(result.Err).
					Str("group", groups[i].Label).
					Msg("failed to update group in SMD")
			}
			errorsOccurred = true
		}
	}
	if errorsOccurred {
		return cli.Errorf(cli.CodeHTTP, "SMD group update completed with errors")
	}

	// Success, log confirmation
	log.Logger.Info().
		Int("group_count", len(groups)).
		Msg("Successfully updated group(s)")

	return nil
}

func newCmdGroupUpdate() *cobra.Command {
	// groupUpdateCmd represents the "smd group update" command
	var groupUpdateCmd = &cobra.Command{
		Use:   "update (-d (<payload_data> | @<payload_file>)) | ([--description <description>] [--tag <tag>]... <group_label>)",
		Args:  cobra.MaximumNArgs(1),
		Short: "Update the description and/or tags of a group",
		Long: `Update the description and/or tags of a group. At least one of --description
or --tag must be specified. Alternatively, pass -d to pass
raw payload data or (if flag argument starts with @) a file
containing the payload data. -f can be specified to change
the format of the input payload data ('json' by default), but
the rules above still apply for the payload. If "-" is used as
the input payload filename, the data is read from standard input.

This command sends a PATCH to SMD. An access token is required.

See ochami-smd(1) for more details.`,
		Example: `  # Update a group using CLI flags
  ochami smd group update --description "New description for compute" compute
  ochami smd group update --tag existing_tag --tag new_tag compute
  ochami smd group update --tag existing_tag,new_tag compute
  ochami smd group update --tag existing_tag,new_tag -D "New description for compute" compute

  # Update groups using input payload data
  ochami smd group update -d '{[
    {
      "label": "compute",
      "description": "New compute group description"
    }
  ]}'

  # Update groups using input payload file
  ochami smd group update -d @payload.json
  ochami smd group update -d @payload.yaml -f yaml

  # Update groups using data from standard input
  echo '<json_data>' | ochami smd group update -d @-
  echo '<yaml_data>' | ochami smd group update -d @- -f yaml`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// cmd.LocalFlags().NFlag() doesn't seem to work, so we check every flag
			if !cmd.Flag("data").Changed {
				if len(args) == 0 {
					return cli.Errorf(cli.CodeUsage, "expected -d or >= 1 argument (group label), got %d", len(args))
				} else {
					if !cmd.Flag("description").Changed && !cmd.Flag("tag").Changed {
						return cli.Errorf(cli.CodeUsage, "group label passed, but no --description/--tag (at least one is required)")
					}
				}
			} else {
				if len(args) > 0 {
					log.Logger.Warn().Msgf("raw data passed, ignoring extra arguments: %v", args)
				}
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Try to get runtime from context (new approach)
			if rt, ok := cli.FromContext(cmd.Context()); ok {
				// Create client to use for requests with runtime
				smdClient, err := smd_lib.GetClientWithRuntime(cmd, rt)
				if err != nil {
					return err
				}

				// Extract options from flags
				// Since flags are registered with the correct types on this command,
				// these Get* calls cannot fail, so we ignore errors with explicit comments
				opts := &groupUpdateOptions{}
				if cmd.Flag("description").Changed {
					opts.Description, _ = cmd.Flags().GetString("description") //nolint:errcheck // Flag registered with matching type, error impossible
				}
				if cmd.Flag("tag").Changed {
					opts.Tags, _ = cmd.Flags().GetStringSlice("tag") //nolint:errcheck // Flag registered with matching type, error impossible
				}

				return runCoreGroupUpdateWithRuntime(cmd, opts, args, smdClient, rt)
			}

			// Fallback to old approach during transition
			// Create client to use for requests
			smdClient, err := smd_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Extract options from flags
			// Since flags are registered with the correct types on this command,
			// these Get* calls cannot fail, so we ignore errors with explicit comments
			opts := &groupUpdateOptions{}
			if cmd.Flag("description").Changed {
				opts.Description, _ = cmd.Flags().GetString("description") //nolint:errcheck // Flag registered with matching type, error impossible
			}
			if cmd.Flag("tag").Changed {
				opts.Tags, _ = cmd.Flags().GetStringSlice("tag") //nolint:errcheck // Flag registered with matching type, error impossible
			}

			return runCoreGroupUpdate(cmd, opts, args, smdClient)
		},
	}

	// Create flags
	groupUpdateCmd.Flags().StringP("description", "D", "", "short description to update group with")
	groupUpdateCmd.Flags().StringSlice("tag", []string{}, "one or more tags to set for group")
	groupUpdateCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	groupUpdateCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	groupUpdateCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)
	groupUpdateCmd.MarkFlagsOneRequired("description", "tag", "data")

	return groupUpdateCmd
}
