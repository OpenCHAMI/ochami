// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package iface

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client/smd"

	smd_lib "github.com/openchami/ochami/internal/cli/smd"
)

// ifaceDeleteOptions holds the flag values for the smd iface delete command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type ifaceDeleteOptions struct {
	All       bool
	NoConfirm bool
}

// runCoreIfaceDeleteWithRuntime contains the core logic for the smd iface delete command.
// It takes the parsed options and performs the actual work of deleting ethernet interfaces.
// Runtime-aware version.
func runCoreIfaceDeleteWithRuntime(cmd *cobra.Command, opts *ifaceDeleteOptions, args []string, smdClient *smd.SMDClient, rt *cli.Runtime) error {
	// Ask before attempting deletion unless --no-confirm was passed
	if !opts.NoConfirm {
		log.Logger.Debug().Msg("--no-confirm not passed, prompting user to confirm deletion")
		var respDelete bool
		var err error
		if opts.All {
			respDelete, err = rt.Ios.LoopYesNo("Really delete ALL ETHERNET INTERFACES?")
		} else {
			respDelete, err = rt.Ios.LoopYesNo("Really delete?")
		}
		if err != nil {
			return cli.Errorf(cli.CodeGeneric, "error fetching user input: %w", err)
		} else if !respDelete {
			log.Logger.Info().Msg("User aborted ethernet interface deletion")
			return nil
		} else {
			log.Logger.Debug().Msg("User answered affirmatively to delete ethernet interfaces")
		}
	}

	// Handle token for this command
	if err := rt.HandleToken(cmd); err != nil {
		return err
	}

	// Create list of ethernet interface IDs to delete
	var eiSlice []smd.EthernetInterface
	var eIdSlice []string
	if cmd.Flag("data").Changed {
		// Use payload file if passed
		if err := rt.HandlePayload(cmd, &eiSlice); err != nil {
			return err
		}
		for _, ei := range eiSlice {
			eIdSlice = append(eIdSlice, ei.ID)
		}
		if len(eIdSlice) == 0 {
			return cli.Errorf(cli.CodeUsage, "payload contained no ethernet interfaces to delete")
		}
	} else {
		// ...otherwise, use passed CLI arguments
		eIdSlice = args
	}

	// Perform deletion
	if opts.All {
		// If --all passed, we don't care about any passed arguments
		_, err := smdClient.DeleteEthernetInterfacesAll(cmd.Context(), rt.Token)
		if err != nil {
			return cli.ClassifyClientError(err,
				"SMD ethernet interface deletion yielded unsuccessful HTTP response",
				"failed to delete ethernet interfaces in SMD")
		}
	} else {
		// If --all not passed, pass argument list to deletion logic
		results := smdClient.DeleteEthernetInterfaces(cmd.Context(), rt.Token, eIdSlice...)
		// Since smdClient.DeleteEthernetInterfaces does the deletion iteratively, we need to deal
		// with each error that might have occurred.
		if err := cli.AggregateItemErrors(results.Errors(), "SMD ethernet interface deletion"); err != nil {
			return err
		}
	}

	return nil
}

func newCmdIfaceDelete() *cobra.Command {
	// ifaceDeleteCmd represents the "smd iface delete" command
	var ifaceDeleteCmd = &cobra.Command{
		Use:   "delete (-d (<payload_data> | @<payload_file>)) | --all | <iface_id>...",
		Short: "Delete one or more ethernet interfaces",
		Long: `Delete one or more ethernet interfaces. These can be specified by one
or more ethernet interface IDs (note this is not the same as a
component xname). Alternatively, pass -d to pass raw payload data
or (if flag argument starts with @) a file containing the
payload data. -f can be specified to change the format of
the input payload data ('json' by default), but the rules
above still apply for the payload. If "-" is used as the
input payload filename, the data is read from standard input.

This command sends a DELETE to SMD. An access token is required.

See ochami-smd(1) for more details.`,
		Example: `  # Delete ethernet interface using CLI flags
  ochami smd iface delete decafc0ffeee
  ochami smd iface delete decafc0ffeee de:ad:be:ee:ee:ef
  ochami smd iface delete --all

  # Delete ethernet interfaces using input payload file
  ochami smd iface delete -d @payload.json
  ochami smd iface delete -d @payload.yaml -f yaml

  # Delete ethernet interfaces using data from standard input
  echo '<json_data>' | ochami smd iface delete -d @-
  echo '<yaml_data>' | ochami smd iface delete -d @- -f yaml`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// With options, only one of:
			// - A payload file with -d
			// - --all
			// - A set of one or more ethernet interface IDs
			// must be passed.
			if !cmd.Flag("all").Changed && !cmd.Flag("data").Changed {
				if len(args) == 0 {
					return cli.Errorf(cli.CodeUsage, "expected -d, --all, or >= 1 argument (interface id), got %d", len(args))
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
			opts := &ifaceDeleteOptions{}
			if cmd.Flag("all").Changed {
				opts.All = true
			}
			if cmd.Flag("no-confirm").Changed {
				opts.NoConfirm, _ = cmd.Flags().GetBool("no-confirm") //nolint:errcheck // Flag registered with matching type, error impossible
			}

			return runCoreIfaceDeleteWithRuntime(cmd, opts, args, smdClient, rt)
		},
	}

	// Create flags
	ifaceDeleteCmd.Flags().BoolP("all", "a", false, "delete all ethernet interfaces in SMD")
	ifaceDeleteCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	ifaceDeleteCmd.Flags().Bool("no-confirm", false, "do not ask before attempting deletion")

	cli.AddFormatInputFlag(ifaceDeleteCmd)
	ifaceDeleteCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return ifaceDeleteCmd
}
