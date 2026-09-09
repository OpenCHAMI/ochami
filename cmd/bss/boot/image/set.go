// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package image

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/openchami/bss/pkg/bssTypes"
	"github.com/spf13/cobra"
	kargs "github.com/synackd/go-kargs"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/format"

	bss_lib "github.com/openchami/ochami/internal/cli/bss"
)

func newCmdBootImageSet() *cobra.Command {
	// bootImageSetCmd represents the "bss boot image set" command
	var bootImageSetCmd = &cobra.Command{
		Use:   "set (-x <xname>[,...] | -m <mac>[,...] | -n <nid>[,...]) <image>",
		Args:  cobra.ExactArgs(1),
		Short: "Set root= kernel command line for one or more nodes, overwriting any previously set",
		Long: `Set root= kernel command line for one or more nodes, overwriting any previously set.
At least one of --xname, --mac, or --nid is required to tell ochami which
components need modification.

An access token is required.

See ochami-bss(1) for more details.`,
		Example: `  # Set nodes to boot live image
  ochami bss boot image set --mac 00:de:ad:be:ef:00,de:ca:fc:0f:fe:ee live:https://172.16.0.254/image.squashfs`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			bssClient, err := bss_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// Get current kernel command line args
			values := url.Values{}
			if cmd.Flag("xname").Changed {
				s, err := cmd.Flags().GetStringSlice("xname")
				if err != nil {
					return cli.Errorf(cli.CodeUsage, "unable to fetch xname list: %w", err)
				}
				for _, x := range s {
					values.Add("name", x)
				}
			}
			if cmd.Flag("mac").Changed {
				s, err := cmd.Flags().GetStringSlice("mac")
				if err != nil {
					return cli.Errorf(cli.CodeUsage, "unable to fetch mac list: %w", err)
				}
				for _, m := range s {
					values.Add("mac", m)
				}
			}
			if cmd.Flag("nid").Changed {
				s, err := cmd.Flags().GetInt32Slice("nid")
				if err != nil {
					return cli.Errorf(cli.CodeUsage, "unable to fetch nid list: %w", err)
				}
				for _, n := range s {
					values.Add("nid", fmt.Sprintf("%d", n))
				}
			}
			qstr := values.Encode()
			httpEnv, err := bssClient.GetBootParams(cmd.Context(), qstr, cli.Token)
			if err != nil {
				return cli.ClassifyClientError(err, "BSS boot parameter request yielded unsuccessful HTTP response", "failed to request boot parameters from BSS")

			}
			var bps []bssTypes.BootParams
			if err := format.UnmarshalData(httpEnv.Body, &bps, format.DataFormatJson); err != nil {
				return cli.Errorf(cli.CodePayload, "failed to unmarshal boot params: %w", err)
			}
			if len(bps) == 0 {
				return cli.Errorf(cli.CodeGeneric, "no boot params to edit")
			}

			// Warn user of any xnames/nids/macs not found
			hostsFound := make(map[string]bool)
			nidsFound := make(map[int32]bool)
			macsFound := make(map[string]bool)
			for _, bp := range bps {
				for _, host := range bp.Hosts {
					hostsFound[host] = true
				}
				for _, nid := range bp.Nids {
					nidsFound[nid] = true
				}
				for _, mac := range bp.Macs {
					macsFound[mac] = true
				}
			}
			if cmd.Flag("xname").Changed {
				s, err := cmd.Flags().GetStringSlice("xname")
				if err != nil {
					return cli.Errorf(cli.CodeUsage, "unable to fetch xname list: %w", err)
				}
				for _, h := range s {
					if _, hFound := hostsFound[h]; !hFound {
						log.Logger.Warn().Msgf("host %s not found, not updating", h)
					}
				}
			}
			if cmd.Flag("nid").Changed {
				s, err := cmd.Flags().GetInt32Slice("nid")
				if err != nil {
					return cli.Errorf(cli.CodeUsage, "unable to fetch nid list: %w", err)
				}
				for _, n := range s {
					if _, nFound := nidsFound[n]; !nFound {
						log.Logger.Warn().Msgf("node ID %d not found, not updating", n)
					}
				}
			}
			if cmd.Flag("mac").Changed {
				s, err := cmd.Flags().GetStringSlice("mac")
				if err != nil {
					return cli.Errorf(cli.CodeUsage, "unable to fetch mac list: %w", err)
				}
				for _, m := range s {
					if _, mFound := macsFound[m]; !mFound {
						log.Logger.Warn().Msgf("mac %s not found, not updating", m)
					}
				}
			}

			errorsOccurred := false
			for bpIdx, bp := range bps {
				// Edit parameters for nodes
				k := kargs.NewKargs([]byte(bp.Params))
				if err := k.SetKarg("root", args[0]); err != nil {
					log.Logger.Error().Err(err).Msg("failed to set 'root' kernel argument")
					errorsOccurred = true
					continue
				}
				bps[bpIdx].Params = k.String()

				// Send modified params back to BSS
				_, err = bssClient.PutBootParams(cmd.Context(), bps[bpIdx], cli.Token)
				if err != nil {
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(err).Msg("BSS boot parameter PUT request yielded unsuccessful HTTP response")
					} else {
						log.Logger.Error().Err(err).Msg("failed to set boot parameters in BSS")
					}
					errorsOccurred = true
				}
			}
			if errorsOccurred {
				return cli.Errorf(cli.CodeHTTP, "updating boot images completed with errors")
			}

			return nil
		},
	}

	// Create flags
	bootImageSetCmd.Flags().StringSliceP("xname", "x", []string{}, "one or more xnames whose boot parameters to set")
	bootImageSetCmd.Flags().StringSliceP("mac", "m", []string{}, "one or more MAC addresses whose boot parameters to set")
	bootImageSetCmd.Flags().Int32SliceP("nid", "n", []int32{}, "one or more node IDs whose boot parameters to set")

	bootImageSetCmd.MarkFlagsOneRequired("xname", "mac", "nid")

	return bootImageSetCmd
}
