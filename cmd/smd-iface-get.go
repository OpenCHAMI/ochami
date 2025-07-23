// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client"
)

// ifaceGetCmd represents the "smd iface get" command
var ifaceGetCmd = &cobra.Command{
	Use:   "get",
	Args:  cobra.NoArgs,
	Short: "Get some or all ethernet interfaces",
	Long: `Get some or all ethernet interfaces optionally based on filter(s). If no options are
passed, all ethernet interfaces are returned. Optionally, options can be passed to limit the
ethernet interfaces returned.

See ochami-smd(1) for more details.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Create client to use for requests
		smdClient := smdGetClient(cmd, false)

		// Deal with --id
		if cmd.Flag("id").Changed {
			// This endpoint requires authentication, so a token is needed
			setTokenFromEnvVar(cmd)
			checkToken(cmd)

			id, err := cmd.Flags().GetString("id")
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to get id")
				logHelpError(cmd)
				os.Exit(1)
			}
			byIP := false
			if cmd.Flag("by-ip").Changed {
				byIP = true
			}
			httpEnv, err := smdClient.GetEthernetInterfaceByID(id, token, byIP)
			if err != nil {
				if errors.Is(err, client.UnsuccessfulHTTPError) {
					log.Logger.Error().Err(err).Msg("SMD ethernet interface request by ID yielded unsuccessful HTTP response")
				} else {
					log.Logger.Error().Err(err).Msg("failed to request ethernet interfaces by ID from SMD")
				}
				logHelpError(cmd)
				os.Exit(1)
			}
			fmt.Println(string(httpEnv.Body))
			os.Exit(0)
		} else if cmd.Flag("by-ip").Changed {
			log.Logger.Error().Msg("--by-ip can only be used with --id")
			logHelpError(cmd)
			os.Exit(1)
		}

		// All other cases
		qstr := ""
		if cmd.Flag("mac").Changed || cmd.Flag("ip").Changed || cmd.Flag("net").Changed || cmd.Flag("comp-id").Changed ||
			cmd.Flag("type").Changed || cmd.Flag("older-than").Changed || cmd.Flag("newer-than").Changed {
			values := url.Values{}
			if cmd.Flag("mac").Changed {
				s, err := cmd.Flags().GetStringSlice("mac")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch macs")
					logHelpError(cmd)
					os.Exit(1)
				}
				for _, m := range s {
					values.Add("MACAddress", m)
				}
			}
			if cmd.Flag("ip").Changed {
				s, err := cmd.Flags().GetStringSlice("ip")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch IPs")
					logHelpError(cmd)
					os.Exit(1)
				}
				for _, i := range s {
					values.Add("IPAddress", i)
				}
			}
			if cmd.Flag("net").Changed {
				s, err := cmd.Flags().GetStringSlice("net")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch networks")
					logHelpError(cmd)
					os.Exit(1)
				}
				for _, n := range s {
					values.Add("Network", n)
				}
			}
			if cmd.Flag("comp-id").Changed {
				s, err := cmd.Flags().GetStringSlice("comp-id")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch component IDs")
					logHelpError(cmd)
					os.Exit(1)
				}
				for _, c := range s {
					values.Add("ComponentID", c)
				}
			}
			if cmd.Flag("type").Changed {
				s, err := cmd.Flags().GetStringSlice("type")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch type")
					logHelpError(cmd)
					os.Exit(1)
				}
				for _, t := range s {
					values.Add("Type", t)
				}
			}
			if cmd.Flag("older-than").Changed {
				s, err := cmd.Flags().GetString("older-than")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch older-than timestamp")
					logHelpError(cmd)
					os.Exit(1)
				}
				values.Add("OlderThan", s)
			}
			if cmd.Flag("newer-than").Changed {
				s, err := cmd.Flags().GetString("newer-than")
				if err != nil {
					log.Logger.Error().Err(err).Msg("unable to fetch newer-than timestamp")
					logHelpError(cmd)
					os.Exit(1)
				}
				values.Add("NewerThan", s)
			}
		}
		httpEnv, err := smdClient.GetEthernetInterfaces(qstr)
		if err != nil {
			if errors.Is(err, client.UnsuccessfulHTTPError) {
				log.Logger.Error().Err(err).Msg("SMD ethernet interface request yielded unsuccessful HTTP response")
			} else {
				log.Logger.Error().Err(err).Msg("failed to request ethernet interfaces from SMD")
			}
			logHelpError(cmd)
			os.Exit(1)
		}

		// Print output
		if outBytes, err := client.FormatBody(httpEnv.Body, formatOutput); err != nil {
			log.Logger.Error().Err(err).Msg("failed to format output")
			logHelpError(cmd)
			os.Exit(1)
		} else {
			fmt.Print(string(outBytes))
		}
	},
}

func init() {
	ifaceGetCmd.Flags().StringP("id", "i", "", "get an ethernet interface by its ID")
	ifaceGetCmd.Flags().Bool("by-ip", false, "get all IP addresses for an ethernet interface (used with --id)")
	ifaceGetCmd.Flags().StringSliceP("mac", "m", []string{}, "filter ethernet interfaces by mac address")
	ifaceGetCmd.Flags().StringSlice("ip", []string{}, "filter ethernet interfaces by IP address")
	ifaceGetCmd.Flags().StringSlice("net", []string{}, "filter ethernet interfaces by IP on given network")
	ifaceGetCmd.Flags().StringSlice("comp-id", []string{}, "filter ethernet interfaces by component ID")
	ifaceGetCmd.Flags().StringSlice("type", []string{}, "filter ethernet interfaces by type")
	ifaceGetCmd.Flags().String("older-than", "", "filter ethernet interfaces by update time older than specified time (RFC3339-formatted)")
	ifaceGetCmd.Flags().String("newer-than", "", "filter ethernet interfaces by update time older than specified time (RFC3339-formatted)")
	ifaceGetCmd.Flags().VarP(&formatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	ifaceGetCmd.RegisterFlagCompletionFunc("format-output", completionFormatData)
	ifaceGetCmd.MarkFlagsMutuallyExclusive("id", "mac")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("id", "ip")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("id", "net")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("id", "comp-id")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("id", "type")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("id", "older-than")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("id", "newer-than")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("by-ip", "mac")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("by-ip", "ip")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("by-ip", "net")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("by-ip", "comp-id")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("by-ip", "type")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("by-ip", "older-than")
	ifaceGetCmd.MarkFlagsMutuallyExclusive("by-ip", "newer-than")

	ifaceCmd.AddCommand(ifaceGetCmd)
}
