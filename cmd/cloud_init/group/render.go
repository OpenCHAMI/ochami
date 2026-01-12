// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"bufio"
	"errors"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/exec"
	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/cloud_init"

	cloud_init_lib "github.com/OpenCHAMI/ochami/internal/cli/cloud_init"
)

func newCmdGroupRender() *cobra.Command {
	// groupRenderCmd represents the "cloud-init group render" command
	var groupRenderCmd = &cobra.Command{
		Use:   "render <group_name> <node_id>",
		Args:  cobra.ExactArgs(2),
		Short: "Render cloud-init config for specific group using a node",
		Long: `Render cloud-init config for specific group using a node.

See ochami-cloud-init(1) for more details.`,
		Example: `  # Render group 'compute' cloud-init config for node x3000c0s0b0n0
  ochami cloud-init group render compute x3000c0s0b0n0`,
		Run: func(cmd *cobra.Command, args []string) {
			// Create client to use for requests
			cloudInitClient := cloud_init_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			// Get group config
			henvs, errs, err := cloudInitClient.GetNodeGroupData(cli.Token, args[1], args[0])
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to get cloud-init group")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
			if errs[0] != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("cloud-init group request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to get cloud-init group")
				}
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
			ciConfigFileBytes := henvs[0].Body

			// Don't try to get meta-data and render if config is empty
			if len(ciConfigFileBytes) == 0 {
				log.Logger.Warn().Msgf("cloud-config for group %s was empty, cannot render for node %s", args[0], args[1])
				os.Exit(0)
			}

			// Get node instance data
			henvs, errs, err = cloudInitClient.GetNodeData(cloud_init.CloudInitMetaData, cli.Token, args[1])
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to get cloud-init node meta-data")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
			if errs[0] != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("cloud-init node meta-data request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to get cloud-init node meta-data")
				}
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
			var ciData map[string]interface{}
			dsWrapper := make(map[string]interface{})
			if err := yaml.Unmarshal(henvs[0].Body, &ciData); err != nil {
				log.Logger.Error().Err(err).Msg("failed to unmarshal HTTP body into map")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
			dsWrapper["ds"] = map[string]interface{}{"meta_data": ciData}
			refData := exec.NewContext(dsWrapper)

			// Render
			tpl, err := gonja.FromBytes(ciConfigFileBytes)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to create template")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}
			out := bufio.NewWriter(os.Stdout)
			if err := tpl.Execute(out, refData); err != nil {
				log.Logger.Error().Err(err).Msg("failed to render template")
				cli.LogHelpError(cmd)
				os.Exit(1)
			}

			// Write rendered template to stdout
			out.Flush()
		},
	}

	return groupRenderCmd
}
