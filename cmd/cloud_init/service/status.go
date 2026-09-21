// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package service

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"

	cloud_init_lib "github.com/openchami/ochami/internal/cli/cloud_init"
)

func newCmdServiceStatus() *cobra.Command {
	// serviceStatusCmd represents the "cloud-init service status" command
	var serviceStatusCmd = &cobra.Command{
		Use:   "status",
		Args:  cobra.NoArgs,
		Short: "Display status of the cloud-init metadata service",
		Long: `Display status of the cloud-init metadata service.

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

			if !cmd.Flag("api").Changed {
				if _, err := cloudInitClient.GetVersion(cmd.Context()); err != nil {
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						if !cmd.Flag("quiet").Changed {
							if err := cli.WriteString(rt.Ios.Out(), "cloud-init is running, but not normally\n"); err != nil {
								return err
							}
						}
						return cli.Errorf(cli.CodeHTTP, "cloud-init status request yielded unsuccessful HTTP response: %w", err)
					}
					if !cmd.Flag("quiet").Changed {
						if err := cli.WriteString(rt.Ios.Out(), "cloud-init is not running\n"); err != nil {
							return err
						}
					}
					return cli.Errorf(cli.CodeNetwork, "failed to get cloud-init status: %w", err)
				}
				if !cmd.Flag("quiet").Changed {
					if err := cli.WriteString(rt.Ios.Out(), "cloud-init is running\n"); err != nil {
						return err
					}
				}
				return nil
			}

			var respArr []client.HTTPEnvelope
			errOccurred := false
			if cmd.Flag("api").Changed {
				if henv, err := cloudInitClient.GetAPI(cmd.Context()); err != nil {
					if errors.Is(err, client.UnsuccessfulHTTPError) {
						rt.Logger.Error().Err(err).Msg("cloud-init API spec request yielded unsuccessful HTTP response")
					} else {
						rt.Logger.Error().Err(err).Msg("failed to get cloud-init API spec")
					}
					errOccurred = true
				} else {
					respArr = append(respArr, henv)
				}
			}

			for _, henv := range respArr {
				outBytes, err := client.FormatBody(henv.Body, rt.FormatOutput)
				if err != nil {
					return cli.Errorf(cli.CodePayload, "failed to format output: %w", err)
				}
				if err := cli.WriteOutput(rt.Ios.Out(), outBytes); err != nil {
					return err
				}
			}

			if errOccurred {
				return cli.Errorf(cli.CodeHTTP, "one or more requests to cloud-init failed")
			}

			return nil
		},
	}

	// Create flags
	serviceStatusCmd.Flags().Bool("api", false, "print OpenAPI spec")
	serviceStatusCmd.Flags().BoolP("quiet", "q", false, "don't print output; return 0 if running, 1 if not")

	serviceStatusCmd.MarkFlagsMutuallyExclusive("quiet", "api")

	cli.AddFormatOutputFlag(serviceStatusCmd)
	serviceStatusCmd.RegisterFlagCompletionFunc("format-output", cli.CompletionFormatData)

	return serviceStatusCmd
}
