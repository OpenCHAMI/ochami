// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rcs

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/rcs"
)

// GetClient sets up the remote-console client with the base URI and certificates
// (if necessary) and returns it. This function is used by each subcommand.
func GetClient(cmd *cobra.Command) (*rcs.RCSClient, error) {
	rcsBaseURI, err := cli.GetBaseURIRCS(cmd)
	if err != nil {
		return nil, cli.Errorf(cli.CodeConfig, "failed to get base URI for remote-console: %w", err)
	}

	insecure, _ := cmd.Flags().GetBool("insecure")

	rcsClient, err := rcs.NewClient(rcsBaseURI, client.WithInsecure(insecure), client.WithShowToken(cli.ShowToken(cmd)))
	if err != nil {
		return nil, cli.Errorf(cli.CodeGeneric, "error creating new remote-console client: %w", err)
	}

	if err := cli.UseCACert(rcsClient.OchamiClient); err != nil {
		return nil, err
	}

	return rcsClient, nil
}
