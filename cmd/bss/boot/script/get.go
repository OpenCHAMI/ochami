// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package script

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"

	bss_lib "github.com/openchami/ochami/internal/cli/bss"
)

func newCmdBootScriptGet() *cobra.Command {
	// bootScriptGetCmd represents the "bss boot script get" command
	var bootScriptGetCmd = &cobra.Command{
		Use:     "get",
		Aliases: []string{"list"},
		Args:    cobra.NoArgs,
		Short:   "Get iPXE boot script for a component",
		Long: `Get iPXE boot script for a component. Specifying one of --mac, --xname,
or --nid is required to specify which component to fetch the boot script for.

This command sends a GET to BSS. An access token is not required.

See ochami-bss(1) for more details.`,
		Example: `  ochami boot script get --mac 00:c0:ff:ee:00:00`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			bssClient, err := bss_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Structure representing the boot script query string
			values := url.Values{}

			// At least one of these required
			if cmd.Flag("xname").Changed {
				s, _ := cmd.Flags().GetStringSlice("xname") //nolint:errcheck // xname is a registered StringSlice flag
				for _, x := range s {
					values.Add("name", x)
				}
			}
			if cmd.Flag("mac").Changed {
				s, _ := cmd.Flags().GetStringSlice("mac") //nolint:errcheck // mac is a registered StringSlice flag
				for _, m := range s {
					values.Add("mac", m)
				}
			}
			if cmd.Flag("nid").Changed {
				s, _ := cmd.Flags().GetInt32Slice("nid") //nolint:errcheck // nid is a registered Int32Slice flag
				for _, n := range s {
					values.Add("nid", fmt.Sprintf("%d", n))
				}
			}

			// These are optional
			if cmd.Flag("retry").Changed {
				s, _ := cmd.Flags().GetInt("retry") //nolint:errcheck // retry is a registered Int flag
				values.Add("retry", fmt.Sprintf("%d", s))
			}
			if cmd.Flag("arch").Changed {
				s, _ := cmd.Flags().GetString("arch") //nolint:errcheck // arch is a registered String flag
				values.Add("arch", s)
			}
			if cmd.Flag("timestamp").Changed {
				s, _ := cmd.Flags().GetInt("timestamp") //nolint:errcheck // timestamp is a registered Int flag
				values.Add("timestamp", fmt.Sprintf("%d", s))
			}
			qstr := values.Encode()

			httpEnv, err := bssClient.GetBootScript(cmd.Context(), qstr)
			if err != nil {
				return cli.ClassifyClientError(err, "BSS boot script request yielded unsuccessful HTTP response", "failed to request boot script from BSS")

			}
			fmt.Fprintln(cli.Ios.Out(), string(httpEnv.Body))

			return nil
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
