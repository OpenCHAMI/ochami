// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/ci"
	"github.com/spf13/cobra"
)

// cloudInitNodeGetCmd represents the "cloud-init group get" command
var cloudInitNodeGetCmd = &cobra.Command{
	Use:   "get",
	Args:  cobra.NoArgs,
	Short: "Get data for specific node(s)",
	Long: `Get data for specific node(s).

See ochami-cloud-init(1) for more details.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		}
	},
}

// cloudInitNodeGetGroupCmd represents the "cloud-init node get group" command
var cloudInitNodeGetGroupCmd = &cobra.Command{
	Use:   "group <id> <group>...",
	Args:  cobra.MinimumNArgs(2),
	Short: "Get group data for a node for one or more groups",
	Long: `Get group data for a node for one or more groups.

See ochami-cloud-init(1) for more details.`,
	Example: `  # Get data from compute and slurm groups for node x3000c0s0b0n0
  ochami cloud-init node get group x3000c0s0b1n0 compute slurm`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Set up cloud-init client
		cloudInitClient := cloudInitGetClient(cmd)

		// This endpoint requires authentication, so a token is needed
		setTokenFromEnvVar(cmd)
		checkToken(cmd)

		// Get node group data
		henvs, errs, err := cloudInitClient.GetNodeGroupData(token, args[0], args[1:]...)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get node group data")
			logHelpError(cmd)
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
			logHelpError(cmd)
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
			if cmd.Flag("no-header").Changed {
				fmt.Println(g)
			} else if cmd.Flag("header").Changed {
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

// cloudInitNodeGetMetadataCmd represents the "cloud-init node get meta-data" command
var cloudInitNodeGetMetadataCmd = &cobra.Command{
	Use:   "meta-data <id>...",
	Args:  cobra.MinimumNArgs(1),
	Short: "Get meta-data for specific node(s)",
	Long: `Get meta-data for specific node(s).

See ochami-cloud-init(1) for more details.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Set up cloud-init client
		cloudInitClient := cloudInitGetClient(cmd)

		// Get meta-data
		henvs, errs, err := cloudInitClient.GetNodeData(ci.CloudInitMetaData, token, args...)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get node meta-data")
			logHelpError(cmd)
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
			logHelpError(cmd)
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
			logHelpError(cmd)
			os.Exit(1)
		}

		// Marshal data into JSON so it can be reformatted into
		// desired output format.
		iiSliceBytes, err := json.Marshal(iiSlice)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to marshal instance info list into JSON")
			logHelpError(cmd)
			os.Exit(1)
		}

		// Print in desired format
		if outBytes, err := client.FormatBody(iiSliceBytes, formatOutput); err != nil {
			log.Logger.Error().Err(err).Msg("failed to format output")
			logHelpError(cmd)
			os.Exit(1)
		} else {
			fmt.Printf(string(outBytes))
		}
	},
}

// cloudInitNodeGetUserdataCmd represents the "cloud-init node get user-data" command
var cloudInitNodeGetUserdataCmd = &cobra.Command{
	Use:   "user-data <id>...",
	Args:  cobra.MinimumNArgs(1),
	Short: "Get user-data for specific node(s)",
	Long: `Get user-data for specific node(s).

See ochami-cloud-init(1) for more details.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Set up cloud-init client
		cloudInitClient := cloudInitGetClient(cmd)

		// Get user-data
		henvs, errs, err := cloudInitClient.GetNodeData(ci.CloudInitUserData, token, args...)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get node user-data")
			logHelpError(cmd)
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
			logHelpError(cmd)
			os.Exit(1)
		}

		// Collect node data into string array
		var iiSlice []string
		for _, henv := range henvs {
			iiSlice = append(iiSlice, string(henv.Body))
		}

		// Print each datum
		for idx, ii := range iiSlice {
			if cmd.Flag("no-header").Changed {
				fmt.Println(ii)
			} else if cmd.Flag("header").Changed {
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

// cloudInitNodeGetVendordataCmd represents the "cloud-init node get vendor-data" command
var cloudInitNodeGetVendordataCmd = &cobra.Command{
	Use:   "vendor-data <id>...",
	Args:  cobra.MinimumNArgs(1),
	Short: "Get vendor-data for specific node(s)",
	Long: `Get vendor-data for specific node(s).

See ochami-cloud-init(1) for more details.`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			printUsageHandleError(cmd)
			os.Exit(0)
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Set up cloud-init client
		cloudInitClient := cloudInitGetClient(cmd)

		// Get vendor-data
		henvs, errs, err := cloudInitClient.GetNodeData(ci.CloudInitVendorData, token, args...)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get node vendor-data")
			logHelpError(cmd)
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
			logHelpError(cmd)
			os.Exit(1)
		}

		// Collect node data into string array
		var iiSlice []string
		for _, henv := range henvs {
			iiSlice = append(iiSlice, string(henv.Body))
		}

		// Print each datum
		for idx, ii := range iiSlice {
			if cmd.Flag("no-header").Changed {
				fmt.Println(ii)
			} else if cmd.Flag("header").Changed {
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

func init() {
	// Add group subcommand
	cloudInitNodeGetGroupCmd.Flags().Bool("no-header", false, "don't print header, even with just one config")
	cloudInitNodeGetGroupCmd.Flags().Bool("header", false, "always print header, even with just one config")
	cloudInitNodeGetCmd.AddCommand(cloudInitNodeGetGroupCmd)

	// Add meta-data subcommand
	cloudInitNodeGetMetadataCmd.PersistentFlags().VarP(&formatOutput, "format-output", "F", "format of output printed to standard output")
	cloudInitNodeGetMetadataCmd.RegisterFlagCompletionFunc("format-output", completionFormatData)
	cloudInitNodeGetCmd.AddCommand(cloudInitNodeGetMetadataCmd)

	// Add user-data subcommand
	cloudInitNodeGetUserdataCmd.Flags().Bool("no-header", false, "don't print header, even with just one config")
	cloudInitNodeGetUserdataCmd.Flags().Bool("header", false, "always print header, even with just one config")
	cloudInitNodeGetCmd.AddCommand(cloudInitNodeGetUserdataCmd)

	// Add vendor-data subcommand
	cloudInitNodeGetVendordataCmd.Flags().Bool("no-header", false, "don't print header, even with just one config")
	cloudInitNodeGetVendordataCmd.Flags().Bool("header", false, "always print header, even with just one config")
	cloudInitNodeGetCmd.AddCommand(cloudInitNodeGetVendordataCmd)

	// Add get command
	cloudInitNodeCmd.AddCommand(cloudInitNodeGetCmd)
}
