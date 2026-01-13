// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package compep

import (
	"errors"
	"fmt"
	"os"

	"github.com/OpenCHAMI/smd/v2/pkg/sm"
	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"

	smd_lib "github.com/OpenCHAMI/ochami/internal/cli/smd"
)

func newCmdCompepDelete() *cobra.Command {
	// compepDeleteCmd represents the "smd compep delete" command
	var compepDeleteCmd = &cobra.Command{
		Use:   "delete (-d (<payload_data> | @<payload_file>)) | --all | <xname>...",
		Short: "Delete one or more component endpoints",
		Long: `Delete one or more component endpoints. These can be specified by one or more xnames.
Alternatively, pass -d to pass raw payload data or (if flag argument
starts with @) a file containing the payload data. -f can be specified
to change the format of the input payload data ('json' by default), but
the rules above still apply for the payload. If "-" is used as the input
payload filename, the data is read from standard input.

This command sends a DELETE to SMD. An access cli.Token is required.

See ochami-smd(1) for more details.`,
		Example: `  # Delete component endpoints using CLI flags
  ochami smd compep delete x3000c1s7b56n0 x3000c1s7b56n1
  ochami smd compep delete --all

  # Delete component endpoints using input payload data
  ochami smd compep delete -d '{"ComponentEndpoints":[{"ID":"x3000c1s7b56n0"},{"ID":"x3000c1s7b56n1"}]}'

  # Delete component endpoints using input payload file
  ochami smd compep delete -d @payload.json
  ochami smd compep delete -d @payload.yaml -f yaml

  # Delete component endpoints using data from standard input
  echo '<json_data>' | ochami smd compep delete -d @-
  echo '<yaml_data>' | ochami smd compep delete -d @- -f yaml`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// With options, only one of:
			// - A payload/file with -d
			// - --all
			// - A set of one or more xnames
			// must be passed.
			if !cmd.Flag("all").Changed && !cmd.Flag("data").Changed {
				if len(args) == 0 {
					return fmt.Errorf("expected -d, --all, or >= 1 argument (xname), got %d", len(args))
				}
			} else {
				if len(args) > 0 {
					log.Logger.Warn().Msgf("raw data or --all passed, ignoring extra arguments: %v", args)
				}
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			smdClient := smd_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Ask before attempting deletion unless --no-confirm was passed
			if !cmd.Flag("no-confirm").Changed {
				log.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
				var respDelete bool
				var err error
				if cmd.Flag("all").Changed {
					respDelete, err = cli.Ios.LoopYesNo("Really delete ALL COMPONENT ENDPOINTS?")
				} else {
					respDelete, err = cli.Ios.LoopYesNo("Really delete?")
				}
				if err != nil {
					log.Logger.Error().Err(err).Msg("Error fetching user input")
					os.Exit(1)
				} else if !respDelete {
					log.Logger.Info().Msg("User aborted component endpoint deletion")
					os.Exit(0)
				} else {
					log.Logger.Debug().Msg("User answered affirmatively to delete component endpoints")
				}
			}

			// Create list of xnames to delete
			var ceSlice []sm.ComponentEndpoint
			var xnameSlice []string
			if cmd.Flag("data").Changed {
				// Use payload file if passed
				cli.HandlePayload(cmd, &ceSlice)
			} else {
				// ...otherwise, use passed CLI arguments
				xnameSlice = args
			}

			// Perform deletion
			if cmd.Flag("all").Changed {
				// If --all passed, we don't care about any passed arguments
				_, err := smdClient.DeleteComponentEndpointsAll(cli.Token)
				if err != nil {
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(err).Msg("SMD component endpoint deletion yielded unsuccessful HTTP response")
					} else {
						log.Logger.Error().Err(err).Msg("failed to delete component endpoints in SMD")
					}
					cli.LogHelpError(cmd)
					os.Exit(1)
				}
			} else {
				// If --all not passed, pass argument list to deletion logic
				_, errs, err := smdClient.DeleteComponentEndpoints(cli.Token, xnameSlice...)
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to delete redfish endpoints in SMD")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}
				// Since smdClient.DeleteComponentEndpoints does the deletion iteratively, we need to
				// deal with each error that might have occurred.
				var errorsOccurred = false
				for _, e := range errs {
					if err != nil {
						if errors.Is(e, client.UnsuccessfulHTTPError) {
							log.Logger.Error().Err(e).Msg("SMD component endpoint deletion yielded unsuccessful HTTP response")
						} else {
							log.Logger.Error().Err(e).Msg("failed to delete component endpoints")
						}
						errorsOccurred = true
					}
				}
				// Warn the user if any errors occurred during deletion iterations
				if errorsOccurred {
					log.Logger.Warn().Msg("SMD component endpoint deletion completed with errors")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}
			}
		},
	}

	// Create flags
	compepDeleteCmd.Flags().BoolP("all", "a", false, "delete all redfish endpoints in SMD")
	compepDeleteCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	compepDeleteCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")
	compepDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	compepDeleteCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return compepDeleteCmd
}
