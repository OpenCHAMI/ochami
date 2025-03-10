// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"

	"github.com/OpenCHAMI/cloud-init/pkg/cistore"
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
		henvs, errs, err := cloudInitClient.GetGroups(args[0])
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
		ciGroupBytes := henvs[0].Body

		// Ensure cloud-config has been decoded
		var (
			ciGroupData       cistore.GroupData
			ciConfigFileBytes []byte
		)
		if err = json.Unmarshal(ciGroupBytes, &ciGroupData); err != nil {
			log.Logger.Error().Err(err).Msg("failed to unmarshal cloud-init group data")
			logHelpError(cmd)
			os.Exit(1)
		}
		if ciConfigFileBytes, err = ci.DecodeCloudConfig(ciGroupData.File); err != nil {
			log.Logger.Error().Err(err).Msg("failed to decode cloud-config file")
			logHelpError(cmd)
			os.Exit(1)
		}

		// Get node instance data
		henvs, errs, err = cloudInitClient.GetNodeData(ci.CloudInitMetaData, args[1])
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
