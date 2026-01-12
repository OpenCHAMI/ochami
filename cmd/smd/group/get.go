// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"

	smd_lib "github.com/OpenCHAMI/ochami/internal/cli/smd"
)

func newCmdGroupGet() *cobra.Command {
	// groupGetCmd represents the "smd group get" command
	var groupGetCmd = &cobra.Command{
		Use:   "get",
		Args:  cobra.NoArgs,
		Short: "Get all groups or group(s) identified by name and/or tag",
		Long: `Get all groups or group(s) identified by name and/or tag.

See ochami-smd(1) for more details.`,
		Example: `  ochami smd group get
  ochami smd group get --name group1
  ochami smd group get --tag group1_tag
  ochami smd group get --name group1,group2
  ochami smd group get --name group1 --name group2
  ochami smd group get --name group1,group2 --tag tag1,tag2
  ochami smd group get --name group1 --name group2 --tag tag1 --tag tag2`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			smdClient := smd_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// If no ID flags are specified, get all groups
			qstr := ""
			if cmd.Flag("name").Changed || cmd.Flag("tag").Changed {
				values := url.Values{}
				if cmd.Flag("name").Changed {
					s, err := cmd.Flags().GetStringSlice("name")
					if err != nil {
						log.Logger.Error().Err(err).Msg("unable to fetch name list")
						cli.LogHelpError(cmd)
						os.Exit(1)
					}
					for _, n := range s {
						values.Add("group", n)
					}
				}
				if cmd.Flag("tag").Changed {
					s, err := cmd.Flags().GetStringSlice("tag")
					if err != nil {
						log.Logger.Error().Err(err).Msg("unable to fetch tag list")
						cli.LogHelpError(cmd)
						os.Exit(1)
					}
					for _, t := range s {
						values.Add("tag", t)
					}
				}
				qstr = values.Encode()
			}
			httpEnv, err := smdClient.GetGroups(qstr, cli.Token)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("SMD group request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to request groups from SMD")
				}
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Print output
			if outBytes, err := client.FormatBody(httpEnv.Body, cli.FormatOutput); err != nil {
				log.Logger.Error().Err(err).Msg("failed to format output")
				cli.LogHelpError(cmd)
				os.Exit(1)
			} else {
				fmt.Print(string(outBytes))
			}
		},
	}

	// Create flags
	groupGetCmd.Flags().StringSlice("name", []string{}, "filter groups by name")
	groupGetCmd.Flags().StringSlice("tag", []string{}, "filter groups by tag")
	groupGetCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	groupGetCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return groupGetCmd
}
