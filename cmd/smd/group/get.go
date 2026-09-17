// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	smd_lib "github.com/openchami/ochami/internal/cli/smd"
)

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
			// Create client to use for requests
			smdClient, err := smd_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// If no ID flags are specified, get all groups
			qstr := ""
			if cmd.Flag("name").Changed || cmd.Flag("tag").Changed {
				values := url.Values{}
				if cmd.Flag("name").Changed {
					s, _ := cmd.Flags().GetStringSlice("name")
					for _, n := range s {
						values.Add("group", n)
					}
				}
				if cmd.Flag("tag").Changed {
					s, _ := cmd.Flags().GetStringSlice("tag")
					for _, t := range s {
						values.Add("tag", t)
					}
				}
				qstr = values.Encode()
			}
			httpEnv, err := smdClient.GetGroups(qstr, cli.Token)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					return cli.Errorf(cli.CodeHTTP, "SMD group request yielded unsuccessful HTTP response: %w", err)
				}
				return cli.Errorf(cli.CodeNetwork, "failed to request groups from SMD: %w", err)
			}

			// Print output
			outBytes, err := client.FormatBody(httpEnv.Body, cli.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
			}
			fmt.Print(string(outBytes))

			return nil
		},
	}

	// Create flags
	groupGetCmd.Flags().StringSlice("name", []string{}, "filter groups by name")
	groupGetCmd.Flags().StringSlice("tag", []string{}, "filter groups by tag")
	groupGetCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	groupGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return groupGetCmd
}
