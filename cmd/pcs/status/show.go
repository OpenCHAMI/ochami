// SPDX-FileCopyrightText: © 2024-2026 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package status

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	pcs_lib "github.com/openchami/ochami/internal/cli/pcs"
	"github.com/openchami/ochami/pkg/format"
)

type statusResponse struct {
	Status []map[string]interface{} `json:"status"`
}

func newCmdStatusShow() *cobra.Command {

	// pcsStatusShowCmd represents the "pcs status show" command
	var pcsStatusShowCmd = &cobra.Command{
		Use:   "show <xname>",
		Args:  cobra.ExactArgs(1),
		Short: "Show power status of target component",
		Long: `Show power status of target component.

See ochami-pcs(1) for more details.`,
		Example: `  # show power status of component
  ochami pcs status show x3000c0s15b0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			xname := args[0]

			// Create client to use for requests
			pcsClient, err := pcs_lib.GetClient(cmd)
			if err != nil {
				return err
			}

			// Handle token for this command
			if err := cli.HandleToken(cmd); err != nil {
				return err
			}

			// Get status
			statusHttpEnv, err := pcsClient.GetStatus([]string{xname}, "", "", cli.Token)
			if err != nil {
				return cli.ClassifyClientError(err, "PCS status request yielded unsuccessful HTTP response", "failed to get power status")

			}

			var output statusResponse

			err = json.Unmarshal(statusHttpEnv.Body, &output)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to unmarshal status: %w", err)
			}

			// Check if status array is empty
			if len(output.Status) == 0 {
				return cli.Errorf(cli.CodeGeneric, "no status found for the specified component")
			}

			// Print output just for first element in status array
			outBytes, err := format.MarshalData(output.Status[0], cli.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
			}
			fmt.Fprintln(cli.Ios.Out(), string(outBytes))

			return nil
		},
	}

	// Define flags
	pcsStatusShowCmd.Flags().VarP(&cli.FormatOutput, "format-output", "F", "format of output printed to standard output (json,json-pretty,yaml)")

	pcsStatusShowCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return pcsStatusShowCmd
}
