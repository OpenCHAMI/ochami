// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"strings"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/ci"
	"github.com/nikolalohinski/gonja/v2"
	"github.com/nikolalohinski/gonja/v2/exec"
	"github.com/spf13/cobra"
)

// cloudInitGroupRenderCmd represents the "cloud-init group render" command
var cloudInitGroupRenderCmd = &cobra.Command{
	Use:   "render <group_name> <node_id>",
	Args:  cobra.ExactArgs(2),
	Short: "Render cloud-init config for specific group using a node",
	Long: `Render cloud-init config for specific group using a node.

See ochami-cloud-init(1) for more details.`,
	Example: `  # Render group 'compute' cloud-init config for node x3000c0s0b0n0
  ochami cloud-init group render compute x3000c0s0b0n0`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Create client to use for requests
		cloudInitClient := cloudInitGetClient(cmd, true)

		// Get group config
		henvs, errs, err := cloudInitClient.GetNodeGroupData(token, args[1], args[0])
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get cloud-init group")
			logHelpError(cmd)
			os.Exit(1)
		}
		if errs[0] != nil {
			if errors.Is(err, client.UnsuccessfulHTTPError) {
				log.Logger.Error().Err(err).Msg("cloud-init group request yielded unsuccessful HTTP response")
			} else {
				log.Logger.Error().Err(err).Msg("failed to get cloud-init group")
			}
			logHelpError(cmd)
			os.Exit(1)
		}
		ciConfigFileBytes := henvs[0].Body

		// Don't try to get meta-data and render if config is empty
		if len(ciConfigFileBytes) == 0 {
			log.Logger.Warn().Msgf("cloud-config for group %s was empty, cannot render for node %s", args[0], args[1])
			os.Exit(0)
		}

		// Get node instance data
		henvs, errs, err = cloudInitClient.GetNodeData(ci.CloudInitMetaData, token, args[1])
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get cloud-init node meta-data")
			logHelpError(cmd)
			os.Exit(1)
		}
		if errs[0] != nil {
			if errors.Is(err, client.UnsuccessfulHTTPError) {
				log.Logger.Error().Err(err).Msg("cloud-init node meta-data request yielded unsuccessful HTTP response")
			} else {
				log.Logger.Error().Err(err).Msg("failed to get cloud-init node meta-data")
			}
			logHelpError(cmd)
			os.Exit(1)
		}
		var ciData map[string]interface{}
		dsWrapper := make(map[string]interface{})
		if err := yaml.Unmarshal(henvs[0].Body, &ciData); err != nil {
			log.Logger.Error().Err(err).Msg("failed to unmarshal HTTP body into map")
			logHelpError(cmd)
			os.Exit(1)
		}
		dsWrapper["ds"] = map[string]interface{}{"meta_data": ciData}

		// Render
		tpl, err := gonja.FromBytes(ciConfigFileBytes)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to create template")
			logHelpError(cmd)
			os.Exit(1)
		}
		tplCtx := exec.NewContext(dsWrapper)
		out, err := tpl.ExecuteToString(tplCtx)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to render template")
			logHelpError(cmd)
			os.Exit(1)
		}

		// Remove '## template: jinja' from first line since already rendered
		trimmedOut := out
		lines := strings.SplitN(out, "\n", 2) // Split only into first line and the rest
		firstLine := strings.TrimRight(lines[0], "\r")
		if firstLine == "## template: jinja" {
			if len(lines) > 1 {
				// Return everything after the first line
				trimmedOut = lines[1]
			}
			// Only one line, and it matched, result is empty
		} else {
			// First line didn't match; return as-is
			trimmedOut = out
		}

		// Print rendered cloud config
		fmt.Println(trimmedOut)
	},
}

func init() {
	cloudInitGroupCmd.AddCommand(cloudInitGroupRenderCmd)
}
