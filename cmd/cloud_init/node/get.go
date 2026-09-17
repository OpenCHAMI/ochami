// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package node

import (
	"encoding/json"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/cloud_init"

	cloud_init_lib "github.com/openchami/ochami/internal/cli/cloud_init"
)

func newCmdNodeGet() *cobra.Command {
	// nodeGetCmd represents the "cloud-init group get" command
	var nodeGetCmd = &cobra.Command{
		Use:     "get",
		Aliases: []string{"list"},
		Args:    cobra.NoArgs,
		Short:   "Get data for specific node(s)",
		Long: `Get data for specific node(s).

See ochami-cloud-init(1) for more details.`,
		RunE: cli.PrintUsage,
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
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Create client to use for requests
			cloudInitClient, err := cloud_init_lib.GetClientWithRuntime(cmd, rt)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := rt.HandleToken(cmd); err != nil {
				return err
			}

			// Get node group data
			results, err := cloudInitClient.GetNodeGroupData(cmd.Context(), rt.Token, args[0], args[1:]...)
			if err != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to get node group data: %w", err)
			}
			// Since the requests are done iteratively, we need to
			// deal with each error that might have occurred.
			var errorsOccurred = false
			for _, e := range results.Errors() {
				if e != nil {
					if errors.Is(e, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(e).Msg("cloud-init node group request yielded unsuccessful HTTP response")
					} else {
						log.Logger.Error().Err(e).Msg("failed to get cloud-init node group data")
					}
					errorsOccurred = true
				}
			}
			if errorsOccurred {
				return cli.Errorf(cli.CodeHTTP, "cloud-init node group data retrieval completed with errors")
			}

			// Collect node group data into string array
			var gSlice []string
			for idx, henv := range results.Values() {
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
					if err := cli.WriteString(rt.Ios.Out(), g+"\n"); err != nil {
						return err
					}
				} else if cloud_init_lib.CIHeaderWhen == cloud_init_lib.CIFlagHeaderAlways {
					if err := cli.WriteString(rt.Ios.Out(), fmt.Sprintf("--- (%d/%d) node=%s group=%s\n%s\n", idx+1, len(gSlice), args[0], args[1+idx], g)); err != nil {
						return err
					}
				} else {
					if len(gSlice) == 1 {
						if err := cli.WriteString(rt.Ios.Out(), g+"\n"); err != nil {
							return err
						}
					} else {
						if err := cli.WriteString(rt.Ios.Out(), fmt.Sprintf("--- (%d/%d) node=%s group=%s\n%s\n", idx+1, len(gSlice), args[0], args[1+idx], g)); err != nil {
							return err
						}
					}
				}
			}

			return nil
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
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Create client to use for requests
			cloudInitClient, err := cloud_init_lib.GetClientWithRuntime(cmd, rt)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := rt.HandleToken(cmd); err != nil {
				return err
			}

			// Get meta-data
			results, err := cloudInitClient.GetNodeData(cmd.Context(), cloud_init.CloudInitMetaData, rt.Token, args...)
			if err != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to get node meta-data: %w", err)
			}
			// Since the requests are done iteratively, we need to
			// deal with each error that might have occurred.
			var errorsOccurred = false
			for _, e := range results.Errors() {
				if e != nil {
					if errors.Is(e, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(e).Msg("cloud-init node meta-data request yielded unsuccessful HTTP response")
					} else {
						log.Logger.Error().Err(e).Msg("failed to get cloud-init node meta-data")
					}
					errorsOccurred = true
				}
			}
			if errorsOccurred {
				return cli.Errorf(cli.CodeHTTP, "cloud-init node meta-data retrieval completed with errors")
			}

			// Collect node data into YAML array
			var iiSlice []map[string]interface{}
			for _, henv := range results.Values() {
				var ii map[string]interface{}
				if err := yaml.Unmarshal(henv.Body, &ii); err != nil {
					return cli.Errorf(cli.CodePayload, "failed to unmarshal HTTP body into group: %w", err)
				}
				iiSlice = append(iiSlice, ii)
			}

			// Marshal data into JSON so it can be reformatted into
			// desired output format.
			iiSliceBytes, err := json.Marshal(iiSlice)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to marshal instance info list into JSON: %w", err)
			}

			// Print in desired format
			outBytes, err := client.FormatBody(iiSliceBytes, rt.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
			}
			if err := cli.WriteOutput(rt.Ios.Out(), outBytes); err != nil {
				return err
			}

			return nil
		},
	}

	// Create flags
	cli.AddFormatOutputFlag(nodeGetMetadataCmd)
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
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Create client to use for requests
			cloudInitClient, err := cloud_init_lib.GetClientWithRuntime(cmd, rt)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := rt.HandleToken(cmd); err != nil {
				return err
			}

			// Get user-data
			results, err := cloudInitClient.GetNodeData(cmd.Context(), cloud_init.CloudInitUserData, rt.Token, args...)
			if err != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to get node user-data: %w", err)
			}
			// Since the requests are done iteratively, we need to
			// deal with each error that might have occurred.
			var errorsOccurred = false
			for _, e := range results.Errors() {
				if e != nil {
					if errors.Is(e, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(e).Msg("cloud-init node user-data request yielded unsuccessful HTTP response")
					} else {
						log.Logger.Error().Err(e).Msg("failed to get cloud-init node user-data")
					}
					errorsOccurred = true
				}
			}
			if errorsOccurred {
				return cli.Errorf(cli.CodeHTTP, "cloud-init node user-data retrieval completed with errors")
			}

			// Collect node data into string array
			var iiSlice []string
			for _, henv := range results.Values() {
				iiSlice = append(iiSlice, string(henv.Body))
			}

			// Print each datum
			for idx, ii := range iiSlice {
				if cloud_init_lib.CIHeaderWhen == cloud_init_lib.CIFlagHeaderNever {
					if err := cli.WriteString(rt.Ios.Out(), ii+"\n"); err != nil {
						return err
					}
				} else if cloud_init_lib.CIHeaderWhen == cloud_init_lib.CIFlagHeaderAlways {
					if err := cli.WriteString(rt.Ios.Out(), fmt.Sprintf("--- (%d/%d) node=%s\n%s\n", idx+1, len(iiSlice), args[idx], ii)); err != nil {
						return err
					}
				} else {
					if len(iiSlice) == 1 {
						if err := cli.WriteString(rt.Ios.Out(), ii+"\n"); err != nil {
							return err
						}
					} else {
						if err := cli.WriteString(rt.Ios.Out(), fmt.Sprintf("--- (%d/%d) node=%s\n%s\n", idx+1, len(iiSlice), args[idx], ii)); err != nil {
							return err
						}
					}
				}
			}

			return nil
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
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Create client to use for requests
			cloudInitClient, err := cloud_init_lib.GetClientWithRuntime(cmd, rt)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := rt.HandleToken(cmd); err != nil {
				return err
			}

			// Get vendor-data
			results, err := cloudInitClient.GetNodeData(cmd.Context(), cloud_init.CloudInitVendorData, rt.Token, args...)
			if err != nil {
				return cli.Errorf(cli.CodeNetwork, "failed to get node vendor-data: %w", err)
			}
			// Since the requests are done iteratively, we need to
			// deal with each error that might have occurred.
			var errorsOccurred = false
			for _, e := range results.Errors() {
				if e != nil {
					if errors.Is(e, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(e).Msg("cloud-init node vendor-data request yielded unsuccessful HTTP response")
					} else {
						log.Logger.Error().Err(e).Msg("failed to get cloud-init node vendor-data")
					}
					errorsOccurred = true
				}
			}
			if errorsOccurred {
				return cli.Errorf(cli.CodeHTTP, "cloud-init node vendor-data retrieval completed with errors")
			}

			// Collect node data into string array
			var iiSlice []string
			for _, henv := range results.Values() {
				iiSlice = append(iiSlice, string(henv.Body))
			}

			// Print each datum
			for idx, ii := range iiSlice {
				if cloud_init_lib.CIHeaderWhen == cloud_init_lib.CIFlagHeaderNever {
					if err := cli.WriteString(rt.Ios.Out(), ii+"\n"); err != nil {
						return err
					}
				} else if cloud_init_lib.CIHeaderWhen == cloud_init_lib.CIFlagHeaderAlways {
					if err := cli.WriteString(rt.Ios.Out(), fmt.Sprintf("--- (%d/%d) node=%s\n%s\n", idx+1, len(iiSlice), args[idx], ii)); err != nil {
						return err
					}
				} else {
					if len(iiSlice) == 1 {
						if err := cli.WriteString(rt.Ios.Out(), ii+"\n"); err != nil {
							return err
						}
					} else {
						if err := cli.WriteString(rt.Ios.Out(), fmt.Sprintf("--- (%d/%d) node=%s\n%s\n", idx+1, len(iiSlice), args[idx], ii)); err != nil {
							return err
						}
					}
				}
			}

			return nil
		},
	}

	// Create flags
	nodeGetVendordataCmd.Flags().Var(&cloud_init_lib.CIHeaderWhen, "headers", "when to print headers above cloud-configs (always,multiple,never")
	nodeGetVendordataCmd.RegisterFlagCompletionFunc("headers", cloud_init_lib.CompletionHeaderWhen)

	return nodeGetVendordataCmd
}
