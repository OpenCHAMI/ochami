// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"os"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
	"github.com/OpenCHAMI/ochami/pkg/client/smd"
	"github.com/spf13/cobra"
)

// ifaceDeleteCmd represents the smd-iface-delete command
var ifaceDeleteCmd = &cobra.Command{
	Use:   "delete -f <payload_file> | --all | <iface_id>...",
	Short: "Delete one or more ethernet interfaces",
	Long: `Delete one or more ethernet interfaces. These can be specified by one or more ethernet
interface IDs (note this is not the same as a component xname). Alternatively,
pass -f to pass a file (optionally specifying --payload-format, JSON by default)
containing the payload data. If - is used as the argument to -f, the data is
read from standard input.

This command sends a DELETE to SMD. An access token is required.`,
	Example: `  ochami smd iface delete decafc0ffeee
  ochami smd iface delete decafc0ffeee de:ad:be:ee:ee:ef
  ochami smd iface delete --all
  ochami smd iface delete -f payload.json
  ochami smd iface delete -f payload.yaml --payload-format yaml
  echo '<json_data>' | ochami smd iface delete -f -
  echo '<yaml_data>' | ochami smd iface delete -f - --payload-format yaml`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// With options, only one of:
		// - A payload file with -f
		// - --all
		// - A set of one or more ethernet interface IDs
		// must be passed.
		if len(args) == 0 {
			if !cmd.Flag("all").Changed && !cmd.Flag("payload").Changed {
				printUsageHandleError(cmd)
				os.Exit(0)
			}
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Without a base URI, we cannot do anything
		smdBaseURI, err := getBaseURISMD(cmd)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for SMD")
			os.Exit(1)
		}

		// This endpoint requires authentication, so a token is needed
		setTokenFromEnvVar(cmd)
		checkToken(cmd)

		// Create client to make request to SMD
		smdClient, err := smd.NewClient(smdBaseURI, insecure)
		if err != nil {
			log.Logger.Error().Err(err).Msg("error creating new SMD client")
			os.Exit(1)
		}

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(smdClient.OchamiClient)

		// Ask before attempting deletion unless --force was passed
		if !cmd.Flag("force").Changed {
			log.Logger.Debug().Msg("--force not passed, prompting user to confirm deletion")
			var respDelete bool
			if cmd.Flag("all").Changed {
				respDelete = loopYesNo("Really delete ALL ETHERNET INTERFACES?")
			} else {
				respDelete = loopYesNo("Really delete?")
			}
			if !respDelete {
				log.Logger.Info().Msg("User aborted ethernet interface deletion")
				os.Exit(0)
			} else {
				log.Logger.Debug().Msg("User answered affirmatively to delete ethernet interfaces")
			}
		}

		// Create list of ethernet interface IDs to delete
		var eiSlice []smd.EthernetInterface
		var eIdSlice []string
		if cmd.Flag("payload").Changed {
			// Use payload file if passed
			handlePayload(cmd, &eiSlice)
		} else {
			// ...otherwise, use passed CLI arguments
			eIdSlice = args
		}

		// Perform deletion
		if cmd.Flag("all").Changed {
			// If --all passed, we don't care about any passed arguments
			_, err := smdClient.DeleteEthernetInterfacesAll(token)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("SMD ethernet interface deletion yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to delete ethernet interfaces in SMD")
				}
				os.Exit(1)
			}
		} else {
			// If --all not passed, pass argument list to deletion logic
			_, errs, err := smdClient.DeleteEthernetInterfaces(token, eIdSlice...)
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to delete ethernet interfaces in SMD")
				os.Exit(1)
			}
			// Since smdClient.DeleteEthernetInterfaces does the deletion iteratively, we need to deal
			// with each error that might have occurred.
			var errorsOccurred = false
			for _, e := range errs {
				if err != nil {
					if errors.Is(e, client.UnsuccessfulHTTPError) {
						log.Logger.Error().Err(e).Msg("SMD ethernet interface deletion yielded unsuccessful HTTP response")
					} else {
						log.Logger.Error().Err(e).Msg("failed to delete ethernet interfaces")
					}
					errorsOccurred = true
				}
			}
			// Warn the user if any errors occurred during deletion iterations
			if errorsOccurred {
				log.Logger.Warn().Msg("SMD ethernet interface deletion completed with errors")
				os.Exit(1)
			}
		}
	},
}

func init() {
	ifaceDeleteCmd.Flags().BoolP("all", "a", false, "delete all ethernet interfaces in SMD")
	ifaceDeleteCmd.Flags().StringP("payload", "f", "", "file containing the request payload; JSON format unless --payload-format specified")
	ifaceDeleteCmd.Flags().StringP("payload-format", "F", defaultPayloadFormat, "format of payload file (yaml,json) passed with --payload")
	ifaceDeleteCmd.Flags().Bool("force", false, "do not ask before attempting deletion")
	ifaceCmd.AddCommand(ifaceDeleteCmd)
}
