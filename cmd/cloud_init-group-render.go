// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/ci"
	"github.com/nikolalohinski/gonja"
	"github.com/spf13/cobra"
)

// cloudInitGroupRenderCmd represents the "cloud-init group render" command
var cloudInitGroupRenderCmd = &cobra.Command{
	Use:   "render <group> <id>",
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
		// Without a base URI, we cannot do anything
		cloudInitbaseURI, err := getBaseURICloudInit(cmd)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for cloud-init")
			logHelpError(cmd)
			os.Exit(1)
		}

		// This endpoint requires authentication, so a token is needed
		setTokenFromEnvVar(cmd)
		checkToken(cmd)

		// Create client to make request to cloud-init
		cloudInitClient, err := ci.NewClient(cloudInitbaseURI, insecure)
		if err != nil {
			log.Logger.Error().Err(err).Msg("error creating new cloud-init client")
			logHelpError(cmd)
			os.Exit(1)
		}

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(cloudInitClient.OchamiClient)

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
		out, err := tpl.Execute(dsWrapper)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to render template")
			logHelpError(cmd)
			os.Exit(1)
		}

		// Print rendered cloud config
		fmt.Println(out)
	},
}

func init() {
	cloudInitGroupCmd.AddCommand(cloudInitGroupRenderCmd)
}
