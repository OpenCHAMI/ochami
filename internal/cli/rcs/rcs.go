// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rcs

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/rcs"
	"github.com/openchami/ochami/pkg/config"
)

// GetClientWithRuntime sets up the remote-console client with the base URI and certificates
// (if necessary) and returns it. This function uses the provided runtime for configuration.
func GetClientWithRuntime(cmd *cobra.Command, rt *cli.Runtime) (*rcs.RCSClient, error) {
	rcsBaseURI, err := rt.GetBaseURI(cmd, config.ServiceRCS)
	if err != nil {
		return nil, cli.Errorf(cli.CodeConfig, "failed to get base URI for remote-console: %w", err)
	}

	insecure, _ := cmd.Flags().GetBool("insecure") //nolint:errcheck // insecure is registered by the RCS parent command

	rcsClient, err := rcs.NewClient(rcsBaseURI, client.WithInsecure(insecure), client.WithShowToken(rt.ShowToken(cmd)))
	if err != nil {
		return nil, cli.Errorf(cli.CodeGeneric, "error creating new remote-console client: %w", err)
	}

	if err := rt.UseCACert(rcsClient.OchamiClient); err != nil {
		return nil, err
	}

	return rcsClient, nil
}

// GetClient sets up the remote-console client with the base URI and certificates
// (if necessary) and returns it. This function uses the runtime from context.
// Since cmd/root.go always injects a runtime into context, this will always succeed.
func GetClient(cmd *cobra.Command) (*rcs.RCSClient, error) {
	// Get runtime from context (always available since cmd/root.go injects it)
	rt, err := cli.RuntimeFromCommand(cmd)
	if err != nil {
		return nil, err
	}
	return GetClientWithRuntime(cmd, rt)
}
