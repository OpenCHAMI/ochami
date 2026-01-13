// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/cloud_init"

	cloud_init_lib "github.com/OpenCHAMI/ochami/internal/cli/cloud_init"
)

func newCmdNodeGet() *cobra.Command {
	// nodeGetCmd represents the "cloud-init group get" command
	var nodeGetCmd = &cobra.Command{
		Use:   "get",
		Args:  cobra.NoArgs,
		Short: "Get data for specific node(s)",
		Long: `Get data for specific node(s).

See ochami-cloud-init(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
				os.Exit(0)
			}
		},
	}

	// Add subcommands
	nodeGetCmd.AddCommand(
		newCmdNodeGetGroup(),
		newCmdNodeGetMetadata(),
		newCmdNodeGetUserdata(),
		newCmdNodeGetVendordata(),
	)

	return nodeGetCmd
}

func newCmdNodeGetGroup() *cobra.Command {
	// nodeGetGroupCmd represents the "cloud-init node get group" command
	var nodeGetGroupCmd = &cobra.Command{
		Use:   "group <node_id> <group_name>...",
		Args:  cobra.MinimumNArgs(2),
		Short: "Get group data for a node for one or more groups",
		Long: `Get group data for a node for one or more groups.

See ochami-cloud-init(1) for more details.`,
		Example: `  # Get data from compute and slurm groups for node x3000c0s0b0n0
  ochami cloud-init node get group x3000c0s0b1n0 compute slurm`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			cloudInitClient := cloud_init_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Get node group data
			henvs, errs, err := cloudInitClient.GetNodeGroupData(cli.Token, args[0], args[1:]...)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to get node group data")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
			// Since the requests are done iteratively, we need to
			// deal with each error that might have occurred.
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(err).Msg("cloud-init node group request yielded unsuccessful HTTP response")
					} else {
						log.Logger.Error().Err(err).Msg("failed to get cloud-init node group data")
					}
					errorsOccurred = true
				}
			}
			if errorsOccurred {
				log.Logger.Warn().Msg("cloud-init node group data retrieval completed with errors")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Collect node group data into string array
			var gSlice []string
			for idx, henv := range henvs {
				// Warn and don't add to list if cloud-config is empty for group
				if len(henv.Body) == 0 {
					log.Logger.Warn().Msgf("cloud-config for group %s was empty, not printing for node %s", args[1+idx], args[0])
					continue
				}
				gSlice = append(gSlice, string(henv.Body))
			}

			// Print each datum
			for idx, g := range gSlice {
				if cloud_init_lib.CIHeaderWhen == cloud_init_lib.CIFlagHeaderNever {
					fmt.Println(g)
				} else if cloud_init_lib.CIHeaderWhen == cloud_init_lib.CIFlagHeaderAlways {
					fmt.Printf("--- (%d/%d) node=%s group=%s\n", idx+1, len(gSlice), args[0], args[1+idx])
					fmt.Println(g)
				} else {
					if len(gSlice) == 1 {
						fmt.Println(g)
					} else {
						fmt.Printf("--- (%d/%d) node=%s group=%s\n", idx+1, len(gSlice), args[0], args[1+idx])
						fmt.Println(g)
					}
				}
			}
		},
	}

	// Create flags
	nodeGetGroupCmd.Flags().Var(&cloud_init_lib.CIHeaderWhen, "headers", "when to print headers above cloud-configs (always,multiple,never")
	nodeGetGroupCmd.RegisterFlagCompletionFunc("headers", cloud_init_lib.CompletionHeaderWhen)

	return nodeGetGroupCmd
}

// nodeGetMetadataCmd represents the "cloud-init node get meta-data" command
func newCmdNodeGetMetadata() *cobra.Command {
	var nodeGetMetadataCmd = &cobra.Command{
		Use:   "meta-data <node_id>...",
		Args:  cobra.MinimumNArgs(1),
		Short: "Get meta-data for specific node(s)",
		Long: `Get meta-data for specific node(s).

See ochami-cloud-init(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			cloudInitClient := cloud_init_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Get meta-data
			henvs, errs, err := cloudInitClient.GetNodeData(cloud_init.CloudInitMetaData, cli.Token, args...)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to get node meta-data")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
			// Since the requests are done iteratively, we need to
			// deal with each error that might have occurred.
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(err).Msg("cloud-init node meta-data request yielded unsuccessful HTTP response")
					} else {
						log.Logger.Error().Err(err).Msg("failed to get cloud-init node meta-data")
					}
					errorsOccurred = true
				}
			}
			if errorsOccurred {
				log.Logger.Warn().Msg("cloud-init node meta-data retrieval completed with errors")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Collect node data into YAML array
			errorsOccurred = false
			var iiSlice []map[string]interface{}
			for _, henv := range henvs {
				var ii map[string]interface{}
				if err := yaml.Unmarshal(henv.Body, &ii); err != nil {
					log.Logger.Error().Err(err).Msg("failed to unmarshal HTTP body into group")
					errorsOccurred = true
				} else {
					iiSlice = append(iiSlice, ii)
				}
			}
			if errorsOccurred {
				log.Logger.Warn().Msg("not all instance info was collected due to errors")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Marshal data into JSON so it can be reformatted into
			// desired output format.
			iiSliceBytes, err := json.Marshal(iiSlice)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to marshal instance info list into JSON")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Print in desired format
			if outBytes, err := client.FormatBody(iiSliceBytes, cli.FormatOutput); err != nil {
				log.Logger.Error().Err(err).Msg("failed to format output")
				cli.LogHelpError(cmd)
				os.Exit(1)
			} else {
				fmt.Print(string(outBytes))
			}
		},
	}

	// Create flags
	nodeGetMetadataCmd.PersistentFlags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output")
	nodeGetMetadataCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return nodeGetMetadataCmd
}

func newCmdNodeGetUserdata() *cobra.Command {
	// nodeGetUserdataCmd represents the "cloud-init node get user-data" command
	var nodeGetUserdataCmd = &cobra.Command{
		Use:   "user-data <node_id>...",
		Args:  cobra.MinimumNArgs(1),
		Short: "Get user-data for specific node(s)",
		Long: `Get user-data for specific node(s).

See ochami-cloud-init(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			cloudInitClient := cloud_init_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Get user-data
			henvs, errs, err := cloudInitClient.GetNodeData(cloud_init.CloudInitUserData, cli.Token, args...)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to get node user-data")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
			// Since the requests are done iteratively, we need to
			// deal with each error that might have occurred.
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(err).Msg("cloud-init node user-data request yielded unsuccessful HTTP response")
					} else {
						log.Logger.Error().Err(err).Msg("failed to get cloud-init node user-data")
					}
					errorsOccurred = true
				}
			}
			if errorsOccurred {
				log.Logger.Warn().Msg("cloud-init node user-data retrieval completed with errors")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Collect node data into string array
			var iiSlice []string
			for _, henv := range henvs {
				iiSlice = append(iiSlice, string(henv.Body))
			}

			// Print each datum
			for idx, ii := range iiSlice {
				if cloud_init_lib.CIHeaderWhen == cloud_init_lib.CIFlagHeaderNever {
					fmt.Println(ii)
				} else if cloud_init_lib.CIHeaderWhen == cloud_init_lib.CIFlagHeaderAlways {
					fmt.Printf("--- (%d/%d) node=%s\n", idx+1, len(iiSlice), args[idx])
					fmt.Println(ii)
				} else {
					if len(iiSlice) == 1 {
						fmt.Println(ii)
					} else {
						fmt.Printf("--- (%d/%d) node=%s\n", idx+1, len(iiSlice), args[idx])
						fmt.Println(ii)
					}
				}
			}
		},
	}

	// Create flags
	nodeGetUserdataCmd.Flags().Var(&cloud_init_lib.CIHeaderWhen, "headers", "when to print headers above cloud-configs (always,multiple,never")
	nodeGetUserdataCmd.RegisterFlagCompletionFunc("headers", cloud_init_lib.CompletionHeaderWhen)

	return nodeGetUserdataCmd
}

func newCmdNodeGetVendordata() *cobra.Command {
	// nodeGetVendordataCmd represents the "cloud-init node get vendor-data" command
	var nodeGetVendordataCmd = &cobra.Command{
		Use:   "vendor-data <node_id>...",
		Args:  cobra.MinimumNArgs(1),
		Short: "Get vendor-data for specific node(s)",
		Long: `Get vendor-data for specific node(s).

See ochami-cloud-init(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			cloudInitClient := cloud_init_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Get vendor-data
			henvs, errs, err := cloudInitClient.GetNodeData(cloud_init.CloudInitVendorData, cli.Token, args...)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to get node vendor-data")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
			// Since the requests are done iteratively, we need to
			// deal with each error that might have occurred.
			var errorsOccurred = false
			for _, err := range errs {
				if err != nil {
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(err).Msg("cloud-init node vendor-data request yielded unsuccessful HTTP response")
					} else {
						log.Logger.Error().Err(err).Msg("failed to get cloud-init node vendor-data")
					}
					errorsOccurred = true
				}
			}
			if errorsOccurred {
				log.Logger.Warn().Msg("cloud-init node vendor-data retrieval completed with errors")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Collect node data into string array
			var iiSlice []string
			for _, henv := range henvs {
				iiSlice = append(iiSlice, string(henv.Body))
			}

			// Print each datum
			for idx, ii := range iiSlice {
				if cloud_init_lib.CIHeaderWhen == cloud_init_lib.CIFlagHeaderNever {
					fmt.Println(ii)
				} else if cloud_init_lib.CIHeaderWhen == cloud_init_lib.CIFlagHeaderAlways {
					fmt.Printf("--- (%d/%d) node=%s\n", idx+1, len(iiSlice), args[idx])
					fmt.Println(ii)
				} else {
					if len(iiSlice) == 1 {
						fmt.Println(ii)
					} else {
						fmt.Printf("--- (%d/%d) node=%s\n", idx+1, len(iiSlice), args[idx])
						fmt.Println(ii)
					}
				}
			}
		},
	}

	// Create flags
	nodeGetVendordataCmd.Flags().Var(&cloud_init_lib.CIHeaderWhen, "headers", "when to print headers above cloud-configs (always,multiple,never")
	nodeGetVendordataCmd.RegisterFlagCompletionFunc("headers", cloud_init_lib.CompletionHeaderWhen)

	return nodeGetVendordataCmd
}
