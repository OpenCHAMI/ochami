// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rfe

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client/smd"

	smd_lib "github.com/openchami/ochami/internal/cli/smd"
)

// rfeDeleteOptions holds the flag values for the smd rfe delete command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type rfeDeleteOptions struct {
	All       bool
	NoConfirm bool
}

// runCoreRfeDeleteWithRuntime contains the core logic for the smd rfe delete command.
// It takes the parsed options and performs the actual work of deleting redfish endpoints.
// Runtime-aware version.
func runCoreRfeDeleteWithRuntime(cmd *cobra.Command, opts *rfeDeleteOptions, args []string, smdClient *smd.SMDClient, rt *cli.Runtime) error {
	// Ask before attempting deletion unless --no-confirm was passed
	if !opts.NoConfirm {
		log.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
		var respDelete bool
		var err error
		if opts.All {
			respDelete, err = rt.Ios.LoopYesNo("Really delete ALL REDFISH ENDPOINTS?")
		} else {
			respDelete, err = rt.Ios.LoopYesNo("Really delete?")
		}
		if err != nil {
			return cli.Errorf(cli.CodeGeneric, "error fetching user input: %w", err)
		} else if !respDelete {
			log.Logger.Info().Msg("User aborted redfish endpoint deletion")
			return nil
		} else {
			log.Logger.Debug().Msg("User answered affirmatively to delete redfish endpoints")
		}
	}

	// Handle token for this command
	if err := rt.HandleToken(cmd); err != nil {
		return err
	}

	// Create list of xnames to delete
	var rfeSlice smd.RedfishEndpointSlice
	var xnameSlice []string
	if cmd.Flag("data").Changed {
		// Use payload file if passed
		if err := rt.HandlePayload(cmd, &rfeSlice); err != nil {
			return err
		}
		for _, rfe := range rfeSlice.RedfishEndpoints {
			xnameSlice = append(xnameSlice, rfe.ID)
		}
		if len(xnameSlice) == 0 {
			return cli.Errorf(cli.CodeUsage, "payload contained no redfish endpoints to delete")
		}
	} else {
		// ...otherwise, use passed CLI arguments
		xnameSlice = args
	}

	// Perform deletion
	if opts.All {
		// If --all passed, we don't care about any passed arguments
		_, err := smdClient.DeleteRedfishEndpointsAll(cmd.Context(), rt.Token)
		if err != nil {
			return cli.ClassifyClientError(err,
				"SMD redfish endpoint deletion yielded unsuccessful HTTP response",
				"failed to delete redfish endpoints in SMD")
		}
	} else {
		// If --all not passed, pass argument list to deletion logic
		results := smdClient.DeleteRedfishEndpoints(cmd.Context(), rt.Token, xnameSlice...)
		// Since smdClient.DeleteRedfishEndpoints does the deletion iteratively, we need to deal with
		// each error that might have occurred.
		if err := cli.AggregateItemErrors(results.Errors(), "SMD redfish endpoint deletion"); err != nil {
			return err
		}
	}

	return nil
}

func newCmdRfeDelete() *cobra.Command {
	// rfeDeleteCmd represents the "smd rfe delete" command
	var rfeDeleteCmd = &cobra.Command{
		Use:   "delete (-d (<payload_data> | @<payload_file>)) | --all | <xname>...",
		Short: "Delete one or more redfish endpoints",
		Long: `Delete one or more redfish endpoints. These can be specified
by one or more xnames. Alternatively, pass -d to pass raw
payload data or (if flag argument starts with @) a file
containing the payload data. -f can be specified to change
the format of the input payload data ('json' by default),
but the rules above still apply for the payload. If "-" is
used as the input payload filename, the data is read from
standard input.

This command sends a DELETE to SMD. An access token is required.

See ochami-smd(1) for more details.`,
		Example: `  # Delete a redfish endpoint using CLI flags
  ochami smd rfe delete x3000c1s7b56
  ochami smd rfe delete x3000c1s7b56 x3000c1s7b56
  ochami smd rfe delete --all

  # Delete redfish endpoints using input payload file
  ochami smd rfe delete -d @payload.json
  ochami smd rfe delete -d @payload.yaml -f yaml

  # Delete redfish endpoints using data from standard input
  echo '<json_data>' | ochami smd rfe delete -d @-
  echo '<yaml_data>' | ochami smd rfe delete -d @- -f yaml`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// With options, only one of:
			// - A payload/file with -d
			// - --all
			// - A set of one or more xnames
			// must be passed.
			if !cmd.Flag("all").Changed && !cmd.Flag("data").Changed {
				if len(args) == 0 {
					return cli.Errorf(cli.CodeUsage, "expected -d, --all, or >= 1 argument (xname), got %d", len(args))
				}
			} else {
				if len(args) > 0 {
					log.Logger.Warn().Msgf("raw data or --all passed, ignoring extra arguments: %v", args)
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
			opts := &rfeDeleteOptions{}
			if cmd.Flag("all").Changed {
				opts.All = true
			}
			if cmd.Flag("no-confirm").Changed {
				opts.NoConfirm, _ = cmd.Flags().GetBool("no-confirm") //nolint:errcheck // Flag registered with matching type, error impossible
			}

			return runCoreRfeDeleteWithRuntime(cmd, opts, args, smdClient, rt)
		},
	}

	// Create flags
	rfeDeleteCmd.Flags().BoolP("all", "a", false, "delete all redfish endpoints in SMD")
	rfeDeleteCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	rfeDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	cli.AddFormatInputFlag(rfeDeleteCmd)
	rfeDeleteCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return rfeDeleteCmd
}
