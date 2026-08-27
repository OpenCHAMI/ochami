// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rfe

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/smd"

	smd_lib "github.com/openchami/ochami/internal/cli/smd"
)

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
			// Ask before attempting deletion unless --no-confirm was passed
			noConfirm, err := cmd.Flags().GetBool("no-confirm")
			if err != nil {
				return cli.Errorf(cli.CodeUsage, "failed to get --no-confirm: %w", err)
			}
			if !noConfirm {
				log.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
				var respDelete bool
				if cmd.Flag("all").Changed {
					respDelete, err = cli.Ios.LoopYesNo("Really delete ALL REDFISH ENDPOINTS?")
				} else {
					respDelete, err = cli.Ios.LoopYesNo("Really delete?")
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

			// Create client to use for requests
			smdClient, err := smd_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// Create list of xnames to delete
			var rfeSlice smd.RedfishEndpointSlice
			var xnameSlice []string
			if cmd.Flag("data").Changed {
				// Use payload file if passed
				if err := cli.HandlePayload(cmd, &rfeSlice); err != nil {
					return err
				}
			} else {
				// ...otherwise, use passed CLI arguments
				xnameSlice = args
			}

			// Perform deletion
			if cmd.Flag("all").Changed {
				// If --all passed, we don't care about any passed arguments
				_, err := smdClient.DeleteRedfishEndpointsAll(cli.Token)
				if err != nil {
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						return cli.Errorf(cli.CodeHTTP, "SMD redfish endpoint deletion yielded unsuccessful HTTP response: %w", err)
					}
					return cli.Errorf(cli.CodeNetwork, "failed to delete redfish endpoints in SMD: %w", err)
				}
			} else {
				// If --all not passed, pass argument list to deletion logic
				_, errs, err := smdClient.DeleteRedfishEndpoints(cli.Token, xnameSlice...)
				if err != nil {
					return cli.Errorf(cli.CodeNetwork, "failed to delete redfish endpoints in SMD: %w", err)
				}
				// Since smdClient.DeleteRedfishEndpoints does the deletion iteratively, we need to deal with
				// each error that might have occurred.
				var errorsOccurred = false
				for _, e := range errs {
					if e != nil {
						if errors.Is(e, client.UnsuccessfulHTTPError) {
							log.Logger.Error().Err(e).Msg("SMD redfish endpoint deletion yielded unsuccessful HTTP response")
						} else {
							log.Logger.Error().Err(e).Msg("failed to delete redfish endpoint")
						}
						errorsOccurred = true
					}
				}
				// Warn the user if any errors occurred during deletion iterations
				if errorsOccurred {
					return cli.Errorf(cli.CodeHTTP, "SMD redfish endpoint deletion completed with errors")
				}
			}

			return nil
		},
	}

	// Create flags
	rfeDeleteCmd.Flags().BoolP("all", "a", false, "delete all redfish endpoints in SMD")
	rfeDeleteCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	rfeDeleteCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")
	rfeDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	rfeDeleteCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return rfeDeleteCmd
}
