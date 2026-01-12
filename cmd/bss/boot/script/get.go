// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package script

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"

	bss_lib "github.com/OpenCHAMI/ochami/internal/cli/bss"
)

func newCmdBootScriptGet() *cobra.Command {
	// bootScriptGetCmd represents the "bss boot script get" command
	var bootScriptGetCmd = &cobra.Command{
		Use:   "get",
		Args:  cobra.NoArgs,
		Short: "Get iPXE boot script for a component",
		Long: `Get iPXE boot script for a component. Specifying one of --mac, --xname,
or --nid is required to specify which component to fetch the boot script for.

This command sends a GET to BSS. An access cli.Token is not required.

See ochami-bss(1) for more details.`,
		Example: `  ochami boot script get --mac 00:c0:ff:ee:00:00`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			bssClient := bss_lib.GetClient(cmd)

			// Structure representing the boot script query string
			values := url.Values{}

			// At least one of these required
			if cmd.Flag("xname").Changed {
				s, err := cmd.Flags().GetStringSlice("xname")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch xname list")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}
				for _, x := range s {
					values.Add("name", x)
				}
			}
			if cmd.Flag("mac").Changed {
				s, err := cmd.Flags().GetStringSlice("mac")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch mac list")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}
				for _, m := range s {
					values.Add("mac", m)
				}
			}
			if cmd.Flag("nid").Changed {
				s, err := cmd.Flags().GetInt32Slice("nid")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch nid list")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}
				for _, n := range s {
					values.Add("nid", fmt.Sprintf("%d", n))
				}
			}

			// These are optional
			if cmd.Flag("retry").Changed {
				s, err := cmd.Flags().GetInt("retry")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch number of retries")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}
				values.Add("retry", fmt.Sprintf("%d", s))
			}
			if cmd.Flag("arch").Changed {
				s, err := cmd.Flags().GetString("arch")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch arch")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}
				values.Add("arch", s)
			}
			if cmd.Flag("timestamp").Changed {
				s, err := cmd.Flags().GetInt("timestamp")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch timestamp")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}
				values.Add("timestamp", fmt.Sprintf("%d", s))
			}
			qstr := values.Encode()

			httpEnv, err := bssClient.GetBootScript(qstr)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("BSS boot script request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to request boot script from BSS")
				}
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
			fmt.Println(string(httpEnv.Body))
		},
	}

	// Create flags
	bootScriptGetCmd.Flags().StringSliceP("xname", "x", []string{}, "one or more xnames whose boot script to get")
	bootScriptGetCmd.Flags().StringSliceP("mac", "m", []string{}, "one or more MAC addresses whose boot script to get")
	bootScriptGetCmd.Flags().Int32SliceP("nid", "n", []int32{}, "one or more node IDs whose boot script to get")
	bootScriptGetCmd.Flags().Int("retry", 0, "number of times to retry fetching boot script on failed boot")
	bootScriptGetCmd.Flags().String("arch", "", "architecture value from iPXE variable ${buildarch}")
	bootScriptGetCmd.Flags().Int("timestamp", 0, "timestamp in seconds since Unix epoch for when SMD state needs to be updated by")

	bootScriptGetCmd.MarkFlagsOneRequired("xname", "mac", "nid")

	return bootScriptGetCmd
}
