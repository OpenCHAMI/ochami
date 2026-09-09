// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/cloud-init/pkg/cistore"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/cloud_init"

	cloud_init_lib "github.com/openchami/ochami/internal/cli/cloud_init"
)

// getGroupData returns a slice of cloud-init group data for the
// requested groups.
func getGroupData(cmd *cobra.Command, args []string) (groupSlice []cistore.GroupData, err error) {
	// Create client to use for requests
	cloudInitClient, err := cloud_init_lib.GetClient(cmd)
	if err != nil {
		return nil, err
	}

	// Handle token for this command
	if err := cli.HandleToken(cmd); err != nil {
		return nil, err
	}

	// Get data
	if len(args) == 0 {
		// No args passed, get all group data at once
		results := cloudInitClient.GetGroups(cmd.Context(), cli.Token)
		if len(results) != 1 {
			return nil, cli.Errorf(cli.CodeNetwork, "cloud-init returned %d results for the all-groups request", len(results))
		}
		if results[0].Err != nil {
			if errors.Is(results[0].Err, client.UnsuccessfulHTTPError) {
				return nil, cli.Errorf(cli.CodeHTTP, "cloud-init group request yielded unsuccessful HTTP response: %w", results[0].Err)
			}
			return nil, cli.Errorf(cli.CodeNetwork, "failed to get cloud-init groups: %w", results[0].Err)
		}

		// Group data is formatted as a map keyed on the name,
		// which is a bit awkward since the name appears twice
		// and is hard to iterate through.
		//
		// Convert group map into group slice.
		var groupMap map[string]cistore.GroupData
		if err := json.Unmarshal(results[0].Value.Body, &groupMap); err != nil {
			return nil, cli.Errorf(cli.CodePayload, "failed to unmarshal all groups: %w", err)
		}
		groupSlice = cloud_init.CIGroupDataMapToSlice(groupMap)
	} else {
		// One or more arguments (group IDs) provided, get data
		// for just those groups.
		results := cloudInitClient.GetGroups(cmd.Context(), cli.Token, args...)
		// Since the requests are done iteratively, we need to
		// deal with each error that might have occurred.
		var errorsOccurred = false
		for _, e := range results.Errors() {
			if e != nil {
				if errors.Is(e, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(e).Msg("cloud-init group request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(e).Msg("failed to get cloud-init groups")
				}
				errorsOccurred = true
			}
		}
		if errorsOccurred {
			return nil, cli.Errorf(cli.CodeHTTP, "cloud-init group retrieval completed with errors")
		}

		// Collect group data into JSON array
		for _, henv := range results.Values() {
			var ciGroup cistore.GroupData
			if err := json.Unmarshal(henv.Body, &ciGroup); err != nil {
				return nil, cli.Errorf(cli.CodePayload, "failed to unmarshal HTTP body into group: %w", err)
			}
			groupSlice = append(groupSlice, ciGroup)
		}
	}
	return groupSlice, nil
}

func newCmdGroupGet() *cobra.Command {
	// groupGetCmd represents the "cloud-init group get" command
	var groupGetCmd = &cobra.Command{
		Use:     "get",
		Aliases: []string{"list"},
		Args:    cobra.NoArgs,
		Short:   "Get group data for all or a subset of cloud-init groups",
		Long: `Get group data for all or a subset of cloud-init groups.

See ochami-cloud-init(1) for more details.`,
		RunE: cli.PrintUsage,
	}

	// Add subcommands
	groupGetCmd.AddCommand(
		newCmdGroupGetConfig(),
		newCmdGroupGetMetadata(),
		newCmdGroupGetRaw(),
	)

	return groupGetCmd
}

func newCmdGroupGetConfig() *cobra.Command {
	// groupGetConfigCmd represents the "cloud-init group get config" command
	var groupGetConfigCmd = &cobra.Command{
		Use:   "config [<group_name>...]",
		Short: "Get cloud-init config from cloud-init server for one or more groups",
		Long: `Get cloud-init config from cloud-init server for one or more groups.

See ochami-cloud-init(1) for more details.`,
		Example: `  # Get just the cloud-init configuration
  ochami cloud-init group get config
  ochami cloud-init group get config compute`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get all data for specified (or unspecified) groups
			groupSlice, err := getGroupData(cmd, args)
			if err != nil {
				return err
			}

			// Extract cloud-config for each group
			type configGroup struct {
				Name     string                 `json:"name" yaml:"name"`
				Data     map[string]interface{} `json:"meta-data" yaml:"meta-data"`
				Content  []byte                 `json:"content" yaml:"content"`
				Encoding string                 `json:"encoding" enums:"base64,plain"`
			}
			var configSlice []configGroup
			for _, config := range groupSlice {
				if len(config.File.Content) == 0 {
					log.Logger.Warn().Msgf("cloud-config for group %s was empty, not printing", config.Name)
					continue
				}
				newCfg := configGroup{
					Name:     config.Name,
					Data:     config.Data,
					Content:  config.File.Content,
					Encoding: config.File.Encoding,
				}

				// Base64 decode any base64-decoded cloud configs
				ccf := cistore.CloudConfigFile{
					Content:  newCfg.Content,
					Encoding: newCfg.Encoding,
				}
				cBytes, err := cloud_init.DecodeCloudConfig(ccf)
				if err != nil {
					return cli.Errorf(cli.CodePayload, "failed to decode cloud-config for %s: %w", newCfg.Name, err)
				}
				newCfg.Content = cBytes
				newCfg.Encoding = "plain"

				configSlice = append(configSlice, newCfg)
			}

			// Print cloud-init config(s)
			for cidx, cfg := range configSlice {
				if cloud_init_lib.CIHeaderWhen == cloud_init_lib.CIFlagHeaderNever {
					fmt.Fprintln(cli.Ios.Out(), string(configSlice[cidx].Content))
				} else if cloud_init_lib.CIHeaderWhen == cloud_init_lib.CIFlagHeaderAlways {
					fmt.Fprintf(cli.Ios.Out(), "--- (%d/%d) group=%s\n", cidx+1, len(configSlice), cfg.Name)
					fmt.Fprintln(cli.Ios.Out(), string(configSlice[cidx].Content))
					fmt.Fprintln(cli.Ios.Out())
				} else {
					if len(configSlice) == 1 {
						fmt.Fprintln(cli.Ios.Out(), string(configSlice[cidx].Content))
					} else {
						fmt.Fprintf(cli.Ios.Out(), "--- (%d/%d) group=%s\n", cidx+1, len(configSlice), cfg.Name)
						fmt.Fprintln(cli.Ios.Out(), string(configSlice[cidx].Content))
					}
				}
			}

			return nil
		},
	}

	// Create flags
	groupGetConfigCmd.Flags().Var(&cloud_init_lib.CIHeaderWhen, "headers", "when to print headers above cloud-configs (always,multiple,never")
	groupGetConfigCmd.RegisterFlagCompletionFunc("headers", cloud_init_lib.CompletionHeaderWhen)

	return groupGetConfigCmd
}

func newCmdGroupGetMetadata() *cobra.Command {
	// groupGetMetaDataCmd represents the "cloud-init group get meta-data" command
	var groupGetMetadataCmd = &cobra.Command{
		Use:   "meta-data [<group_name>...]",
		Short: "Get meta-data from cloud-init server for one or more groups",
		Long: `Get meta-data from cloud-init server for one or more groups.

See ochami-cloud-init(1) for more details.`,
		Example: `  # Get just the meta-data
  ochami cloud-init group get meta-data
  ochami cloud-init group get meta-data compute`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get all data for specified (or unspecified) groups
			groupSlice, err := getGroupData(cmd, args)
			if err != nil {
				return err
			}

			// Extract meta-data for each group
			type mdGroup struct {
				Name string                 `json:"name" yaml:"name"`
				Data map[string]interface{} `json:"meta-data" yaml:"meta-data"`
			}
			var mdSlice []mdGroup
			for _, group := range groupSlice {
				newGr := mdGroup{
					Name: group.Name,
					Data: group.Data,
				}
				mdSlice = append(mdSlice, newGr)
			}

			// Marshal data into JSON so it can be reformatted into
			// desired output format.
			groupSliceBytes, err := json.Marshal(mdSlice)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to marshal group list into JSON: %w", err)
			}

			// Print in desired format
			outBytes, err := client.FormatBody(groupSliceBytes, cli.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
			}
			fmt.Fprint(cli.Ios.Out(), string(outBytes))

			return nil
		},
	}

	// Create flags
	groupGetMetadataCmd.PersistentFlags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output")
	groupGetMetadataCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return groupGetMetadataCmd
}

func newCmdGroupGetRaw() *cobra.Command {
	// groupGetRawCmd represents the "cloud-init group get raw" command
	var groupGetRawCmd = &cobra.Command{
		Use:   "raw [<group_name>...]",
		Short: "Get raw data from cloud-init server for one or more groups",
		Long: `Get raw data from cloud-init server for one or more groups.

See ochami-cloud-init(1) for more details.`,
		Example: `  # Get raw information about group from cloud-init server
  ochami cloud-init group get raw
  ochami cloud-init group get raw compute`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get all data for specified (or unspecified) groups
			groupSlice, err := getGroupData(cmd, args)
			if err != nil {
				return err
			}

			// Marshal data into JSON so it can be reformatted into
			// desired output format.
			groupSliceBytes, err := json.Marshal(groupSlice)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to marshal group list into JSON: %w", err)
			}

			// Print in desired format
			outBytes, err := client.FormatBody(groupSliceBytes, cli.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
			}
			fmt.Fprint(cli.Ios.Out(), string(outBytes))

			return nil
		},
	}

	// Create flags
	groupGetRawCmd.PersistentFlags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output")
	groupGetRawCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return groupGetRawCmd
}
