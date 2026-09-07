// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package component

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client/smd"

	smd_lib "github.com/openchami/ochami/internal/cli/smd"
)

func newCmdComponentDelete() *cobra.Command {
	// componentDeleteCmd represents the "smd component delete" command
	var componentDeleteCmd = &cobra.Command{
		Use:   "delete (-d (<payload_data> | @<payload_file>)) | --all | <xname>...",
		Short: "Delete one or more components",
		Long: `Delete one or more components. These can be specified by one or more xnames, one
or more NIDs, or a combination of both. Alternatively,
pass -d to pass raw payload data or (if flag argument
starts with @) a file containing the payload data. -f
can be specified to change the format of the input
payload data ('json' by default), but the rules above
still apply for the payload. If "-" is used as the input
payload filename, the data is read from standard input.

This command sends a DELETE to SMD. An access token is required.

See ochami-smd(1) for more details.`,
		Example: `  # Delete components using CLI flags
  ochami smd component delete x3000c1s7b56n0
  ochami smd component delete x3000c1s7b56n0 x3000c1s7b56n1
  ochami smd component delete --all

  # Delete components using input payload data
  ochami smd component delete -d '{"Components":[{"ID"x3000c1s7b56n0"},{"ID":"x3000c1s7b56n1"}]}'

  # Delete components using input payload file
  ochami smd component delete -d @payload.json
  ochami smd component delete -d @payload.yaml -f yaml

  # Delete components using data from standard input
  echo '<json_data>' | ochami smd component delete -d @-
  echo '<yaml_data>' | ochami smd component delete -d @- -f yaml`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// With options, only one of:
			// - A payload file with -d
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
					respDelete, err = cli.Ios.LoopYesNo("Really delete ALL COMPONENTS?")
				} else {
					respDelete, err = cli.Ios.LoopYesNo("Really delete?")
				}
				if err != nil {
					return cli.Errorf(cli.CodeGeneric, "error fetching user input: %w", err)
				} else if !respDelete {
					log.Logger.Info().Msg("User aborted component deletion")
					return nil
				} else {
					log.Logger.Debug().Msg("User answered affirmatively to delete components")
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
			var compSlice smd.ComponentSlice
			var xnameSlice []string
			if cmd.Flag("data").Changed {
				// Use payload file if passed
				if err := cli.HandlePayload(cmd, &compSlice); err != nil {
					return err
				}
				for _, component := range compSlice.Components {
					xnameSlice = append(xnameSlice, component.ID)
				}
				if len(xnameSlice) == 0 {
					return cli.Errorf(cli.CodeUsage, "payload contained no components to delete")
				}
			} else {
				// ...otherwise, use passed CLI arguments
				xnameSlice = args
			}

			// Perform deletion
			if cmd.Flag("all").Changed {
				// If --all passed, we don't care about any passed arguments
				_, err := smdClient.DeleteComponentsAll(cli.Token)
				if err != nil {
					return cli.ClassifyClientError(err,
						"SMD component deletion yielded unsuccessful HTTP response",
						"failed to delete components in SMD")
				}
			} else {
				// If --all not passed, pass argument list to deletion logic
				_, errs, err := smdClient.DeleteComponents(cli.Token, xnameSlice...)
				if err != nil {
					return cli.Errorf(cli.CodeNetwork, "failed to delete components in SMD: %w", err)
				}
				// Since smdClient.DeleteComponents does the deletion iteratively, we need to deal with
				// each error that might have occurred.
				if err := cli.AggregateItemErrors(errs, "SMD component deletion"); err != nil {
					return err
				}
			}

			return nil
		},
	}

	// Create flags
	componentDeleteCmd.Flags().BoolP("all", "a", false, "delete all components in SMD")
	componentDeleteCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	componentDeleteCmd.Flags().VarP(&cli.FormatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")
	componentDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	componentDeleteCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return componentDeleteCmd
}
