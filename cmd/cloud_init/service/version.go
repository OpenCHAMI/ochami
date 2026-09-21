// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package service

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	cloud_init_lib "github.com/openchami/ochami/internal/cli/cloud_init"
)

func newCmdServiceVersion() *cobra.Command {
	// serviceVersionCmd represents the "cloud-init service status" command
	var serviceVersionCmd = &cobra.Command{
		Use:   "version",
		Args:  cobra.NoArgs,
		Short: "Print version of the cloud-init metadata service",
		Long: `Print version of the cloud-init metadata service.

See ochami-cloud-init(1) for more details.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get runtime from context (always available since cmd/root.go injects it)
			rt, err := cli.RuntimeFromCommand(cmd)
			if err != nil {
				return err
			}

			// Create client to use for requests
			cloudInitClient, err := cloud_init_lib.GetClientWithRuntime(cmd, rt)
			if err != nil {
				return err
			}

			henv, err := cloudInitClient.GetVersion(cmd.Context())
			if err != nil {
				return cli.ClassifyClientError(err, "cloud-init version request yielded unsuccessful HTTP response", "failed to get cloud-init version")

			}

			outBytes, err := client.FormatBody(henv.Body, rt.FormatOutput)
			if err != nil {
				return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
			}
			if err := cli.WriteOutput(rt.Ios.Out(), outBytes); err != nil {
				return err
			}

			return nil
		},
	}

	// Create flags

	cli.AddFormatOutputFlag(serviceVersionCmd)
	serviceVersionCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return serviceVersionCmd
}
