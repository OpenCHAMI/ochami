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
	"github.com/OpenCHAMI/ochami/pkg/client/smd"
)

// componentAddCmd represents the "smd component add" command
var componentAddCmd = &cobra.Command{
	Use:   "add (-d (<payload_data> | @<payload_file>)) | (<xname> <node_id>)",
	Short: "Add new component(s)",
	Long: `Add new component(s). A name (xname) and node ID (int64) are
required. Alternatively, pass -d to pass raw payload data
or (if flag argument starts with @) a file containing the
payload data. -f can be specified to change the format of
the input payload data ('json' by default), but the rules
above still apply for the payload. If "-" is used as the
input payload filename, the data is read from standard input.

This command sends a POST to SMD. An access token is required.

See ochami-smd(1) for more details.`,
	Example: `  # Add component using CLI flags
  ochami smd component add x3000c1s7b56n0 56
  ochami smd component add --state Ready --enabled --role Compute --arch X86 x3000c1s7b56n0 56

  # Add components using input payload data
  ochami smd component add -d '{
    "Components":[
      {
        "ID": "x3000c1s7b56n0",
	"NID": 56,
	"State": "Ready",
	"Role": "Compute",
	"Enabled": true,
	"Arch": "X86"
      }
    ]
  }'

  # Add components using input payload file
  ochami smd component add -d @payload.json
  ochami smd component add -d @payload.yaml -f yaml

  # Add components using data from standard input
  echo '<json_data>' | ochami smd component add -d @-
  echo '<yaml_data>' | ochami smd component add -d @- -f yaml`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Check that all required args are passed
		if !cmd.Flag("data").Changed {
			if len(args) == 0 {
				return fmt.Errorf("expected -d or 2 arguments (xname, nid), got neither")
			} else if len(args) != 2 {
				return fmt.Errorf("expected -d or 2 arguments (xname, nid) but got %d: %v", len(args), args)
			}
		} else {
			if len(args) > 0 {
				log.Logger.Warn().Msgf("raw data passed, ignoring extra arguments: %v", args)
			}
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Create client to use for requests
		smdClient := smdGetClient(cmd)

		// Handle token for this command
		handleToken(cmd)

		var compSlice smd.ComponentSlice
		var err error
		if cmd.Flag("data").Changed {
			handlePayload(cmd, &compSlice)
		} else {
			// ...otherwise use CLI options
			comp := smd.Component{
				ID:    args[0],
				State: cmd.Flag("state").Value.String(),
				Role:  cmd.Flag("role").Value.String(),
				Arch:  cmd.Flag("arch").Value.String(),
			}
			comp.Enabled, err = cmd.Flags().GetBool("enabled")
			if err != nil {
				log.Logger.Error().Err(err).Msg("failed to retrieve flag 'enabled', defaulting to true")
				comp.Enabled = true
			}

			compSlice.Components = append(compSlice.Components, comp)
		}

		// Send off request
		_, err = smdClient.PostComponents(compSlice, token)
		if err != nil {
			if errors.Is(err, client.UnsuccessfulHTTPError) {
				log.Logger.Error().Err(err).Msg("SMD component request yielded unsuccessful HTTP response")
			} else {
				log.Logger.Error().Err(err).Msg("failed to add component(s) to SMD")
			}
			logHelpError(cmd)
			os.Exit(1)
		}
	},
}

func init() {
	componentAddCmd.Flags().String("state", "Ready", "set readiness state of new component")
	componentAddCmd.Flags().Bool("enabled", true, "set if new component is enabled")
	componentAddCmd.Flags().String("role", "Compute", "role of new component")
	componentAddCmd.Flags().String("arch", "X86", "CPU architecture of new component")
	componentAddCmd.Flags().StringP("data", "d", "", "payload data or (if starting with @) file containing payload data (can be - to read from stdin)")
	componentAddCmd.Flags().VarP(&formatInput, "format-input", "f", "format of input payload data (json,json-pretty,yaml)")

	componentAddCmd.RegisterFlagCompletionFunc("format-input", completionFormatData)

	componentAddCmd.MarkFlagsMutuallyExclusive("state", "data")
	componentAddCmd.MarkFlagsMutuallyExclusive("enabled", "data")
	componentAddCmd.MarkFlagsMutuallyExclusive("role", "data")
	componentAddCmd.MarkFlagsMutuallyExclusive("arch", "data")

	componentCmd.AddCommand(componentAddCmd)
}
