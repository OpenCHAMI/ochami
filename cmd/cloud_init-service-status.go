// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
)

// cloudInitServiceStatusCmd represents the "cloud-init service status" command
var cloudInitServiceStatusCmd = &cobra.Command{
	Use:   "status",
	Args:  cobra.NoArgs,
	Short: "Check/Manage the cloud-init metadata service",
	Long: `Check/Manage the cloud-init metadata service. This is a metacommand.

See ochami-cloud-init(1) for more details.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create client to use for requests
		cloudInitClient := cloudInitGetClient(cmd)

		if !cmd.Flag("version").Changed && !cmd.Flag("api").Changed {
			if _, err := cloudInitClient.GetVersion(); err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("cloud-init status request yielded unsuccessful HTTP response")
					if !cmd.Flag("quiet").Changed {
						fmt.Println("cloud-init is running, but not normally")
					}
					os.Exit(1)
				} else {
					log.Logger.Error().Err(err).Msg("failed to get cloud-init status")
					if !cmd.Flag("quiet").Changed {
						fmt.Println("cloud-init is not running")
					}
					os.Exit(1)
				}
			} else {
				if !cmd.Flag("quiet").Changed {
					fmt.Println("cloud-init is running")
				}
				os.Exit(0)
			}
		}

		var respArr []client.HTTPEnvelope
		errOccurred := false
		if cmd.Flag("version").Changed {
			if henv, err := cloudInitClient.GetVersion(); err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("cloud-init version request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to get cloud-init version")
				}
				errOccurred = true
			} else {
				respArr = append(respArr, henv)
			}
		}
		if cmd.Flag("api").Changed {
			if henv, err := cloudInitClient.GetAPI(); err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("cloud-init API spec request yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to get cloud-init API spec")
				}
				errOccurred = true
			} else {
				respArr = append(respArr, henv)
			}
		}

		for _, henv := range respArr {
			if outBytes, err := client.FormatBody(henv.Body, formatOutput); err != nil {
				log.Logger.Error().Err(err).Msg("failed to format output")
				logHelpError(cmd)
				os.Exit(1)
			} else {
				fmt.Print(string(outBytes))
			}
		}

		if errOccurred {
			log.Logger.Warn().Msg("one or more requests to cloud-init failed")
			os.Exit(1)
		}
	},
}

func init() {
	cloudInitServiceStatusCmd.Flags().Bool("api", false, "print OpenAPI spec")
	cloudInitServiceStatusCmd.Flags().BoolP("quiet", "q", false, "don't print output; return 0 if running, 1 if not")
	cloudInitServiceStatusCmd.Flags().Bool("version", false, "print version information of cloud-init")
	cloudInitServiceStatusCmd.Flags().VarP(&formatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	cloudInitServiceStatusCmd.MarkFlagsMutuallyExclusive("quiet", "api")
	cloudInitServiceStatusCmd.MarkFlagsMutuallyExclusive("quiet", "version")

	cloudInitServiceStatusCmd.RegisterFlagCompletionFunc("format-output", completionFormatData)

	cloudInitServiceCmd.AddCommand(cloudInitServiceStatusCmd)
}
