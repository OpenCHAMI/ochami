// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/cloud-init/pkg/cistore"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/cloud_init"

	cloud_init_lib "github.com/OpenCHAMI/ochami/internal/cli/cloud_init"
)

// getGroupData returns a slice of cloud-init group data for the
// requested groups. If an error occurs, the program exits.
func getGroupData(cmd *cobra.Command, args []string) (groupSlice []cistore.GroupData) {
	// Create client to use for requests
	cloudInitClient := cloud_init_lib.GetClient(cmd)

	// Handle token for this command
	cli.HandleToken(cmd)

	// Get data
	if len(args) == 0 {
		// No args passed, get all group data at once
		henvs, errs, err := cloudInitClient.GetGroups(cli.Token)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get all groups from cloud-init")
			cli.LogHelpError(cmd)
			os.Exit(1)
		}
		if errs[0] != nil {
			if errors.Is(errs[0], client.UnsuccessfulHTTPError) {
				log.Logger.Error().Err(errs[0]).Msg("cloud-init group request yielded unsuccessful HTTP response")
			} else {
				log.Logger.Error().Err(errs[0]).Msg("failed to cloud-init groups")
			}
			cli.LogHelpError(cmd)
			os.Exit(1)
		}

		// Group data is formatted as a map keyed on the name,
		// which is a bit awkward since the name appears twice
		// and is hard to iterate through.
		//
		// Convert group map into group slice.
		var groupMap map[string]cistore.GroupData
		if err := json.Unmarshal(henvs[0].Body, &groupMap); err != nil {
			log.Logger.Error().Err(err).Msg("failed to unmarshal all groups")
			cli.LogHelpError(cmd)
			os.Exit(1)
		}
		groupSlice = cloud_init.CIGroupDataMapToSlice(groupMap)
	} else {
		// One or more arguments (group IDs) provided, get data
		// for just those groups.
		henvs, errs, err := cloudInitClient.GetGroups(cli.Token, args...)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get cloud-init groups")
			cli.LogHelpError(cmd)
			os.Exit(1)
		}
		// Since the requests are done iteratively, we need to
		// deal with each error that might have occurred.
		var errorsOccurred = false
		for _, err := range errs {
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("cloud-init group request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to get cloud-init groups")
				}
				errorsOccurred = true
			}
		}
		if errorsOccurred {
			log.Logger.Warn().Msg("cloud-init group retrieval completed with errors")
			cli.LogHelpError(cmd)
			os.Exit(1)
		}

		// Collect group data into JSON array
		errorsOccurred = false
		for _, henv := range henvs {
			var ciGroup cistore.GroupData
			if err := json.Unmarshal(henv.Body, &ciGroup); err != nil {
				log.Logger.Error().Err(err).Msg("failed to unmarshal HTTP body into group")
				errorsOccurred = true
			} else {
				groupSlice = append(groupSlice, ciGroup)
			}
		}
		if errorsOccurred {
			log.Logger.Warn().Msg("not all group data was collected due to errors")
			cli.LogHelpError(cmd)
			os.Exit(1)
		}
	}
	return
}

func newCmdGroupGet() *cobra.Command {
	// groupGetCmd represents the "cloud-init group get" command
	var groupGetCmd = &cobra.Command{
		Use:   "get",
		Args:  cobra.NoArgs,
		Short: "Get group data for all or a subset of cloud-init groups",
		Long: `Get group data for all or a subset of cloud-init groups.

See ochami-cloud-init(1) for more details.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cli.PrintUsageHandleError(cmd)
				os.Exit(0)
			}
		},
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
		Run: func(cmd *cobra.Command, args []string) {
			// Get all data for specified (or unspecified) groups
			groupSlice := getGroupData(cmd, args)

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
				if cBytes, err := cloud_init.DecodeCloudConfig(ccf); err != nil {
					log.Logger.Error().Err(err).Msgf("failed to decode cloud-config for %s", newCfg.Name)
					cli.LogHelpError(cmd)
					os.Exit(1)
				} else {
					newCfg.Content = cBytes
					newCfg.Encoding = "plain"
				}

				configSlice = append(configSlice, newCfg)
			}

			// Print cloud-init config(s)
			for cidx, cfg := range configSlice {
				if cloud_init_lib.CIHeaderWhen == cloud_init_lib.CIFlagHeaderNever {
					fmt.Println(string(configSlice[cidx].Content))
				} else if cloud_init_lib.CIHeaderWhen == cloud_init_lib.CIFlagHeaderAlways {
					fmt.Printf("--- (%d/%d) group=%s\n", cidx+1, len(configSlice), cfg.Name)
					fmt.Println(string(configSlice[cidx].Content))
					fmt.Println()
				} else {
					if len(configSlice) == 1 {
						fmt.Println(string(configSlice[cidx].Content))
					} else {
						fmt.Printf("--- (%d/%d) group=%s\n", cidx+1, len(configSlice), cfg.Name)
						fmt.Println(string(configSlice[cidx].Content))
					}
				}
			}
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
		Run: func(cmd *cobra.Command, args []string) {
			// Get all data for specified (or unspecified) groups
			groupSlice := getGroupData(cmd, args)

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
				log.Logger.Error().Err(err).Msg("failed to marshal group list into JSON")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Print in desired format
			if outBytes, err := client.FormatBody(groupSliceBytes, cli.FormatOutput); err != nil {
				log.Logger.Error().Err(err).Msg("failed to format output")
				cli.LogHelpError(cmd)
				os.Exit(1)
			} else {
				fmt.Print(string(outBytes))
			}
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
		Run: func(cmd *cobra.Command, args []string) {
			// Get all data for specified (or unspecified) groups
			groupSlice := getGroupData(cmd, args)

			// Marshal data into JSON so it can be reformatted into
			// desired output format.
			groupSliceBytes, err := json.Marshal(groupSlice)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to marshal group list into JSON")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Print in desired format
			if outBytes, err := client.FormatBody(groupSliceBytes, cli.FormatOutput); err != nil {
				log.Logger.Error().Err(err).Msg("failed to format output")
				cli.LogHelpError(cmd)
				os.Exit(1)
			} else {
				fmt.Print(string(outBytes))
			}
		},
	}

	// Create flags
	groupGetRawCmd.PersistentFlags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output")
	groupGetRawCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return groupGetRawCmd
}
