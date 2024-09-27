// Copyright © 2024 Triad National Security, LLC. All rights reserved.
//
// This program was produced under U.S. Government contract 89233218CNA000001
// for Los Alamos National Laboratory (LANL), which is operated by Triad
// National Security, LLC for the U.S. Department of Energy/National Nuclear
// Security Administration. All rights in the program are reserved by Triad
// National Security, LLC, and the U.S. Department of Energy/National Nuclear
// Security Administration. The Government is granted for itself and others
// acting on its behalf a nonexclusive, paid-up, irrevocable worldwide license
// in this material to reproduce, prepare derivative works, distribute copies to
// the public, perform publicly and display publicly, and to permit others to do
// so.
package cmd

import (
	"errors"
	"os"

	"github.com/OpenCHAMI/bss/pkg/bssTypes"
	"github.com/spf13/cobra"
	"github.com/synackd/ochami/internal/client"
	"github.com/synackd/ochami/internal/log"
)

// bssBootParamsAddCmd represents the bss-bootparams-add command
var bssBootParamsAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add new boot parameters for one or more components",
	Run: func(cmd *cobra.Command, args []string) {
		// cmd.LocalFlags().NFlag() doesn't seem to work, so we check every flag
		if len(args) == 0 &&
			!cmd.Flag("xname").Changed && !cmd.Flag("nid").Changed && !cmd.Flag("mac").Changed &&
			!cmd.Flag("kernel").Changed && !cmd.Flag("initrd").Changed {
			err := cmd.Usage()
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to print usage")
				os.Exit(1)
			}
			os.Exit(0)
		}

		// Without a base URI, we cannot do anything
		bssBaseURI, err := getBaseURI(cmd)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for BSS")
			os.Exit(1)
		}

		// This endpoint requires authentication, so a token is needed
		// TODO: Check token validity/expiration
		checkToken(cmd)

		// Create client to make request to BSS
		bssClient, err := client.NewBSSClient(bssBaseURI, insecure)
		if err != nil {
			log.Logger.Error().Err(err).Msg("error creating new BSS client")
			os.Exit(1)
		}

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(bssClient.OchamiClient)

		// The BSS BootParams struct we will send
		bp := bssTypes.BootParams{}

		// Set the hosts the boot parameters are for
		if cmd.Flag("xname").Changed {
			bp.Hosts, err = cmd.Flags().GetStringSlice("xname")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to fetch xname list")
				os.Exit(1)
			}
		}
		if cmd.Flag("mac").Changed {
			bp.Macs, err = cmd.Flags().GetStringSlice("mac")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to fetch mac list")
				os.Exit(1)
			}
			if err = bp.CheckMacs(); err != nil {
				log.Logger.Error().Err(err).Msg("invalid mac(s)")
				os.Exit(1)
			}
		}
		if cmd.Flag("nid").Changed {
			bp.Nids, err = cmd.Flags().GetInt32Slice("nid")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to fetch nid list")
				os.Exit(1)
			}
		}

		// Set the boot parameters
		if cmd.Flag("kernel").Changed {
			bp.Kernel, err = cmd.Flags().GetString("kernel")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to fetch kernel uri")
				os.Exit(1)
			}
		}
		if cmd.Flag("initrd").Changed {
			bp.Initrd, err = cmd.Flags().GetString("initrd")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to fetch initrd uri")
				os.Exit(1)
			}
		}
		if cmd.Flag("params").Changed {
			bp.Params, err = cmd.Flags().GetString("params")
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to fetch params")
				os.Exit(1)
			}
		}

		// Send 'em off
		_, err = bssClient.PostBootParams(bp, token)
		if err != nil {
			if errors.Is(err, client.UnsuccessfulHTTPError) {
				log.Logger.Error().Err(err).Msg("BSS boot parameter request yielded unsuccessful HTTP response")
			} else {
				log.Logger.Error().Err(err).Msg("failed to add boot parameters to BSS")
			}
			os.Exit(1)
		}
	},
}

func init() {
	bssBootParamsAddCmd.Flags().String("kernel", "", "URI of kernel")
	bssBootParamsAddCmd.Flags().String("initrd", "", "URI of initrd/initramfs")
	bssBootParamsAddCmd.Flags().String("params", "", "kernel parameters")
	bssBootParamsAddCmd.Flags().StringSliceP("xname", "x", []string{}, "one or more xnames whose boot parameters to add")
	bssBootParamsAddCmd.Flags().StringSliceP("mac", "m", []string{}, "one or more MAC addresses whose boot parameters to add")
	bssBootParamsAddCmd.Flags().Int32SliceP("nid", "n", []int32{}, "one or more node IDs whose boot parameters to add")

	bssBootParamsAddCmd.MarkFlagsOneRequired("xname", "mac", "nid")
	bssBootParamsAddCmd.MarkFlagsOneRequired("kernel", "initrd", "params")

	bssBootParamsCmd.AddCommand(bssBootParamsAddCmd)
}