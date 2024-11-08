// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/OpenCHAMI/ochami/internal/client"
	"github.com/OpenCHAMI/ochami/internal/log"
)

// ifaceDeleteCmd represents the iface-delete command
var ifaceDeleteCmd = &cobra.Command{
	Use:   "delete -f <payload_file> | --all | <iface_id>...",
	Short: "Delete one or more ethernet interfaces",
	Long: `Delete one or more ethernet interfaces. These can be specified by one or more ethernet
interface IDs (note this is not the same as a component xname).

This command sends a DELETE to SMD. An access token is required.`,
	Example: `  ochami iface delete decafc0ffeee
  ochami iface delete decafc0ffeee de:ad:be:ee:ee:ef
  ochami iface delete --all
  ochami iface delete -f payload.json
  ochami iface delete -f payload.yaml --payload-format yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		// With options, only one of:
		// - A payload file with -f
		// - --all
		// - A set of one or more ethernet interface IDs
		// must be passed.
		if len(args) == 0 {
			if !cmd.Flag("all").Changed && !cmd.Flag("payload").Changed {
				err := cmd.Usage()
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to print usage")
					os.Exit(1)
				}
				os.Exit(0)
			}
		}

		// Without a base URI, we cannot do anything
		smdBaseURI, err := getBaseURI(cmd)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for SMD")
			os.Exit(1)
		}

		// This endpoint requires authentication, so a token is needed
		setTokenFromEnvVar(cmd)
		checkToken(cmd)

		// Create client to make request to SMD
		smdClient, err := client.NewSMDClient(smdBaseURI, insecure)
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
		var eiSlice []client.EthernetInterface
		var eIdSlice []string
		if cmd.Flag("payload").Changed {
			// Use payload file if passed
			dFile := cmd.Flag("payload").Value.String()
			dFormat := cmd.Flag("payload-format").Value.String()
			err := client.ReadPayload(dFile, dFormat, &eiSlice)
			if err != nil {
				log.Logger.Error().Err(err).Msg("unable to read payload for request")
				os.Exit(1)
			}
			for _, ei := range eiSlice {
				eIdSlice = append(eIdSlice, ei.ID)
			}
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
				if errors.Is(e, client.UnsuccessfulHTTPError) {
					errorsOccurred = true
					log.Logger.Error().Err(e).Msg("SMD ethernet interface deletion yielded unsuccessful HTTP response")
				} else if e != nil {
					errorsOccurred = true
					log.Logger.Error().Err(e).Msg("failed to delete ethernet interfaces")
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
	ifaceDeleteCmd.Flags().String("payload-format", defaultPayloadFormat, "format of payload file (yaml,json) passed with --payload")
	ifaceDeleteCmd.Flags().Bool("force", false, "do not ask before attempting deletion")
	ifaceCmd.AddCommand(ifaceDeleteCmd)
}
