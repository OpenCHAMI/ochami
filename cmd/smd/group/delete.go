// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client/smd"

	smd_lib "github.com/openchami/ochami/internal/cli/smd"
)

// groupDeleteOptions holds the flag values for the smd group delete command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type groupDeleteOptions struct {
	NoConfirm bool
}

// runCoreGroupDeleteWithRuntime contains the core logic for the smd group delete command.
// It takes the parsed options and performs the actual work of deleting groups.
// Runtime-aware version.
func runCoreGroupDeleteWithRuntime(cmd *cobra.Command, opts *groupDeleteOptions, args []string, smdClient *smd.SMDClient, rt *cli.Runtime) error {
	// Ask before attempting deletion unless --no-confirm was passed
	if !opts.NoConfirm {
		log.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
		respDelete, err := rt.Ios.LoopYesNo("Really delete?")
		if err != nil {
			return cli.Errorf(cli.CodeGeneric, "error fetching user input: %w", err)
		} else if !respDelete {
			log.Logger.Info().Msg("User aborted group deletion")
			return nil
		} else {
			log.Logger.Debug().Msg("User answered affirmatively to delete groups")
		}
	}

	// Handle token for this command
	if err := rt.HandleToken(cmd); err != nil {
		return err
	}

	// Create list of group labels to delete
	var groups []smd.Group
	var gLabelSlice []string
	if cmd.Flag("data").Changed {
		// Use payload file if passed
		if err := rt.HandlePayload(cmd, &groups); err != nil {
			return err
		}
		for _, group := range groups {
			gLabelSlice = append(gLabelSlice, group.Label)
		}
		if len(gLabelSlice) == 0 {
			return cli.Errorf(cli.CodeUsage, "payload contained no groups to delete")
		}
	} else {
		// ...otherwise, use passed CLI arguments
		gLabelSlice = args
	}

	// Perform deletion
	results := smdClient.DeleteGroups(cmd.Context(), rt.Token, gLabelSlice...)
	// Since smdClient.DeleteGroups does the deletion iteratively, we need to deal with
	// each error that might have occurred.
	if err := cli.AggregateItemErrors(results.Errors(), "SMD group deletion"); err != nil {
		return err
	}

	return nil
}

func newCmdGroupDelete() *cobra.Command {
	// groupDeleteCmd represents the "smd group delete" command
	var groupDeleteCmd = &cobra.Command{
		Use:   "delete (-d (<payload_data> | @<payload_file>)) | <group_label>...",
		Short: "Delete one or more groups",
		Long: `Delete one or more groups. These can be specified by one or more group labels.
Alternatively, pass -d to pass raw payload data or (if flag
argument starts with @) a file containing the payload data.
-f can be specified to change the format of the input payload
data ('json' by default), but the rules above still apply for
the payload. If "-" is used as the input payload filename, the
data is read from standard input.

This command sends a DELETE to SMD. An access token is required.

See ochami-smd(1) for more details.`,
		Example: `  # Delete groups using CLI flags
  ochami smd group delete compute

  # Delete groups using input payload data
  ochami smd group delete -d '{[{"label":"compute"}]}'

  # Delete groups using input payload file
  ochami smd group delete -d @payload.json
  ochami smd group delete -d @payload.yaml -f yaml

  # Delete groups using data from standard input
  echo '<json_data>' | ochami smd group delete -d @-
  echo '<yaml_data>' | ochami smd group delete -d @- -f yaml`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// With options, only one of:
			// - A payload file with -d
			// - A set of one or more group labels
			// must be passed.
			if !cmd.Flag("data").Changed {
				if len(args) == 0 {
					return cli.Errorf(cli.CodeUsage, "expected -d or >= 1 argument (group label), got %d", len(args))
				}
			} else {
				if len(args) > 1 {
					log.Logger.Warn().Msgf("raw data passed, ignoring extra arguments: %v", args)
				}
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Create client to use for requests with runtime
			smdClient, err := smd_lib.GetClientWithRuntime(cmd, rt)
			if err != nil {
				return err
			}

			// Extract options from flags
			// Since flags are registered with the correct types on this command,
			// these Get* calls cannot fail, so we ignore errors with explicit comments
			opts := &groupDeleteOptions{}
			if cmd.Flag("no-confirm").Changed {
				opts.NoConfirm, _ = cmd.Flags().GetBool("no-confirm") //nolint:errcheck // Flag registered with matching type, error impossible
			}

			return runCoreGroupDeleteWithRuntime(cmd, opts, args, smdClient, rt)
		},
	}

	// Create flags
	groupDeleteCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	groupDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	cli.AddFormatInputFlag(groupDeleteCmd)
	groupDeleteCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return groupDeleteCmd
}
