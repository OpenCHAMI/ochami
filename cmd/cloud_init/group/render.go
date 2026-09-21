// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package group

import (
	"gopkg.in/yaml.v3"

	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/exec"
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/cloud_init"

	cloud_init_lib "github.com/openchami/ochami/internal/cli/cloud_init"
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
  ochami cloud-init group render compute x3000c0s0b0n0

  # Render group 'compute' cloud-init config for node x1000c0s0b0n0, loading extra variables in
  # from extra-vars.json, stdin, and directly, respectively
  ochami -k cloud-init group render --extra-vars @extra-vars.json compute x1000c0s0b0n0
  ochami -k cloud-init group render --extra-vars @- compute x1000c0s0b0n0
  ochami -k cloud-init group render --extra-vars '{"key":"value"}' compute x1000c0s0b0n0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Create client to use for requests
			cloudInitClient, err := cloud_init_lib.GetClient(cmd, rt)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := rt.HandleToken(cmd); err != nil {
				return err
			}

			// Get group config
			results, err := cloudInitClient.GetNodeGroupData(cmd.Context(), rt.Token, args[1], args[0])
			if err != nil {
				return cli.ClassifyClientError(err, "failed to get cloud-init group", "failed to get cloud-init group")
			}
			if results[0].Err != nil {
				return cli.ClassifyClientError(results[0].Err, "cloud-init group request yielded unsuccessful HTTP response", "failed to get cloud-init group")
			}
			ciConfigFileBytes := results[0].Value.Body

			// Don't try to get meta-data and render if config is empty
			if len(ciConfigFileBytes) == 0 {
				rt.Logger.Warn().Msgf("cloud-config for group %s was empty, cannot render for node %s", args[0], args[1])
				return nil
			}

			// Get node instance data
			results, err = cloudInitClient.GetNodeData(cmd.Context(), cloud_init.CloudInitMetaData, rt.Token, args[1])
			if err != nil {
				return cli.ClassifyClientError(err, "failed to get cloud-init node meta-data", "failed to get cloud-init node meta-data")
			}
			if results[0].Err != nil {
				return cli.ClassifyClientError(results[0].Err, "cloud-init node meta-data request yielded unsuccessful HTTP response", "failed to get cloud-init node meta-data")
			}
			var ciData map[string]interface{}
			dsWrapper := make(map[string]interface{})
			if err := yaml.Unmarshal(results[0].Value.Body, &ciData); err != nil {
				return cli.Errorf(cli.CodePayload, "failed to unmarshal HTTP body into map: %w", err)
			}
			dsWrapper["ds"] = map[string]interface{}{"meta_data": ciData}

			// Read any extra variables specified (This is mostly copy-pasted from rt.HandlePayload)
			// The primary difference is the flag name
			extraVarsMap := make(map[string]interface{})
			if cmd.Flag("extra-vars").Changed {
				extraVars := cmd.Flag("extra-vars").Value.String()
				if err := client.ReadPayloadWithReader(extraVars, rt.Ios.In(), rt.FormatInput, &extraVarsMap); err != nil {
					return cli.Errorf(cli.CodePayload, "unable to read extra variable data or file: %w", err)
				}
			}

			// Apply extra variables to the context
			for k, v := range extraVarsMap {
				dsWrapper[k] = v
			}

			// Construct the context for the template
			refData := exec.NewContext(dsWrapper)

			// Render
			tpl, err := gonja.FromBytes(ciConfigFileBytes)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to create template: %w", err)
			}
			if err := tpl.Execute(rt.Ios.Out(), refData); err != nil {
				return cli.Errorf(cli.CodePayload, "failed to render template: %w", err)
			}

			return nil
		},
	}

	// Create flags
	groupRenderCmd.Flags().StringP("extra-vars", "e", "", "extra variables to be passed to the template renderer or (if starting with @) file containing extra variables (can be - to read from stdin)")

	cli.AddFormatInputFlag(groupRenderCmd)
	groupRenderCmd.RegisterFlagCompletionFunc("format-input", cli.CompletionFormatData)

	return groupRenderCmd
}
