// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	smd_lib "github.com/openchami/ochami/internal/cli/smd"
	"github.com/openchami/ochami/pkg/client/smd"
)

// groupGetOptions holds the flag values for the smd group get command.
// This struct is used to avoid ignored errors by providing a clean interface
// for accessing flag values that are registered with the correct types.
type groupGetOptions struct {
	Names []string
	Tags  []string
}

// runCoreGroupGet contains the core logic for the smd group get command.
// It takes the parsed options and performs the actual work of getting groups.
// Runtime-aware version.
func runCoreGroupGetWithRuntime(cmd *cobra.Command, opts *groupGetOptions, smdClient *smd.SMDClient, rt *cli.Runtime) error {
	// Handle token for this command
	if err := rt.HandleToken(cmd); err != nil {
		return err
	}

	// If no ID flags are specified, get all groups
	qstr := ""
	if len(opts.Names) > 0 || len(opts.Tags) > 0 {
		values := url.Values{}
		for _, n := range opts.Names {
			values.Add("group", n)
		}
		for _, t := range opts.Tags {
			values.Add("tag", t)
		}
		qstr = values.Encode()
	}

	httpEnv, err := smdClient.GetGroups(cmd.Context(), qstr, rt.Token)
	if err != nil {
		return cli.ClassifyClientError(err, "SMD group request yielded unsuccessful HTTP response", "failed to request groups from SMD")
	}

	// Print output
	outBytes, err := client.FormatBody(httpEnv.Body, rt.FormatOutput)
	if err != nil {
		return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
	}
	if err := cli.WriteOutput(rt.Ios.Out(), outBytes); err != nil {
		return err
	}

	return nil
}

func newCmdGroupGet() *cobra.Command {
	// groupGetCmd represents the "smd group get" command
	var groupGetCmd = &cobra.Command{
		Use:     "get",
		Aliases: []string{"list"},
		Args:    cobra.NoArgs,
		Short:   "Get all groups or group(s) identified by name and/or tag",
		Long: `Get all groups or group(s) identified by name and/or tag.

See ochami-smd(1) for more details.`,
		Example: `  ochami smd group get
  ochami smd group get --name group1
  ochami smd group get --tag group1_tag
  ochami smd group get --name group1,group2
  ochami smd group get --name group1 --name group2
  ochami smd group get --name group1,group2 --tag tag1,tag2
  ochami smd group get --name group1 --name group2 --tag tag1 --tag tag2`,
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
			opts := &groupGetOptions{}
			if cmd.Flag("name").Changed {
				opts.Names, _ = cmd.Flags().GetStringSlice("name") //nolint:errcheck // Flag registered with matching type, error impossible
			}
			if cmd.Flag("tag").Changed {
				opts.Tags, _ = cmd.Flags().GetStringSlice("tag") //nolint:errcheck // Flag registered with matching type, error impossible
			}

			return runCoreGroupGetWithRuntime(cmd, opts, smdClient, rt)
		},
	}

	// Create flags
	groupGetCmd.Flags().StringSlice("name", []string{}, "filter groups by name")
	groupGetCmd.Flags().StringSlice("tag", []string{}, "filter groups by tag")

	cli.AddFormatOutputFlag(groupGetCmd)
	groupGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return groupGetCmd
}
