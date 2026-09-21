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
	headerWhen := cloud_init_lib.CIFlagHeaderWhen(cloud_init_lib.CIFlagHeaderMultiple)

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
			// Create client to use for requests
			cloudInitClient, err := cloud_init_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// Get node group data
			results, err := cloudInitClient.GetNodeGroupData(cmd.Context(), cli.Token, args[0], args[1:]...)
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

			// Collect node group data for rendering.
			var items []cloud_init_lib.RenderItem
			for idx, henv := range results.Values() {
				// Warn and don't add to list if cloud-config is empty for group
				if len(henv.Body) == 0 {
					log.Logger.Warn().Msgf("cloud-config for group %s was empty, not printing for node %s", args[1+idx], args[0])
					continue
				}
				items = append(items, cloud_init_lib.RenderItem{
					Labels: fmt.Sprintf("node=%s group=%s", args[0], args[1+idx]),
					Body:   string(henv.Body),
				})
			}

			if err := cloud_init_lib.Render(cli.Ios.Out(), headerWhen, items); err != nil {
				return cli.Errorf(cli.CodeGeneric, "failed to write cloud-init node group data: %w", err)
			}

			return nil
		},
	}

	// Create flags
	nodeGetGroupCmd.Flags().Var(&headerWhen, "headers", "when to print headers above cloud-configs (always,multiple,never")
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
			// Create client to use for requests
			cloudInitClient, err := cloud_init_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// Get meta-data
			results, err := cloudInitClient.GetNodeData(cmd.Context(), cloud_init.CloudInitMetaData, cli.Token, args...)
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
			errorsOccurred = false
			for _, henv := range results.Values() {
				var ii map[string]interface{}
				if err := yaml.Unmarshal(henv.Body, &ii); err != nil {
					log.Logger.Error().Err(err).Msg("failed to unmarshal HTTP body into group")
					errorsOccurred = true
				} else {
					iiSlice = append(iiSlice, ii)
				}
			}
			if errorsOccurred {
				return cli.Errorf(cli.CodePayload, "not all instance info was collected due to errors")
			}

			// Marshal data into JSON so it can be reformatted into
			// desired output format.
			iiSliceBytes, err := json.Marshal(iiSlice)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to marshal instance info list into JSON: %w", err)
			}

			// Print in desired format
			outBytes, err := client.FormatBody(iiSliceBytes, cli.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
			}
			fmt.Fprint(cli.Ios.Out(), string(outBytes))

			return nil
		},
	}

	// Create flags
	nodeGetMetadataCmd.PersistentFlags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output")
	nodeGetMetadataCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return nodeGetMetadataCmd
}

func newCmdNodeGetUserdata() *cobra.Command {
	headerWhen := cloud_init_lib.CIFlagHeaderWhen(cloud_init_lib.CIFlagHeaderMultiple)

	// nodeGetUserdataCmd represents the "cloud-init node get user-data" command
	var nodeGetUserdataCmd = &cobra.Command{
		Use:   "user-data <node_id>...",
		Args:  cobra.MinimumNArgs(1),
		Short: "Get user-data for specific node(s)",
		Long: `Get user-data for specific node(s).

See ochami-cloud-init(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			cloudInitClient, err := cloud_init_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// Get user-data
			results, err := cloudInitClient.GetNodeData(cmd.Context(), cloud_init.CloudInitUserData, cli.Token, args...)
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

			// Collect node data for rendering.
			items := make([]cloud_init_lib.RenderItem, 0, len(results))
			for idx, henv := range results.Values() {
				items = append(items, cloud_init_lib.RenderItem{
					Labels: fmt.Sprintf("node=%s", args[idx]),
					Body:   string(henv.Body),
				})
			}

			if err := cloud_init_lib.Render(cli.Ios.Out(), headerWhen, items); err != nil {
				return cli.Errorf(cli.CodeGeneric, "failed to write cloud-init node user-data: %w", err)
			}

			return nil
		},
	}

	// Create flags
	nodeGetUserdataCmd.Flags().Var(&headerWhen, "headers", "when to print headers above cloud-configs (always,multiple,never")
	nodeGetUserdataCmd.RegisterFlagCompletionFunc("headers", cloud_init_lib.CompletionHeaderWhen)

	return nodeGetUserdataCmd
}

func newCmdNodeGetVendordata() *cobra.Command {
	headerWhen := cloud_init_lib.CIFlagHeaderWhen(cloud_init_lib.CIFlagHeaderMultiple)

	// nodeGetVendordataCmd represents the "cloud-init node get vendor-data" command
	var nodeGetVendordataCmd = &cobra.Command{
		Use:   "vendor-data <node_id>...",
		Args:  cobra.MinimumNArgs(1),
		Short: "Get vendor-data for specific node(s)",
		Long: `Get vendor-data for specific node(s).

See ochami-cloud-init(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create client to use for requests
			cloudInitClient, err := cloud_init_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// Get vendor-data
			results, err := cloudInitClient.GetNodeData(cmd.Context(), cloud_init.CloudInitVendorData, cli.Token, args...)
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

			// Collect node data for rendering.
			items := make([]cloud_init_lib.RenderItem, 0, len(results))
			for idx, henv := range results.Values() {
				items = append(items, cloud_init_lib.RenderItem{
					Labels: fmt.Sprintf("node=%s", args[idx]),
					Body:   string(henv.Body),
				})
			}

			if err := cloud_init_lib.Render(cli.Ios.Out(), headerWhen, items); err != nil {
				return cli.Errorf(cli.CodeGeneric, "failed to write cloud-init node vendor-data: %w", err)
			}

			return nil
		},
	}

	// Create flags
	nodeGetVendordataCmd.Flags().Var(&headerWhen, "headers", "when to print headers above cloud-configs (always,multiple,never")
	nodeGetVendordataCmd.RegisterFlagCompletionFunc("headers", cloud_init_lib.CompletionHeaderWhen)

	return nodeGetVendordataCmd
}
