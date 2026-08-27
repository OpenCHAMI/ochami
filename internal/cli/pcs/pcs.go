// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package pcs

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/pcs"
)

// GetClient sets up the PCS client with the PCS base URI and certificates
// (if necessary) and returns it. This function is used by each subcommand.
func GetClient(cmd *cobra.Command) (*pcs.PCSClient, error) {
	// Without a base URI, we cannot do anything
	pcsBaseURI, err := cli.GetBaseURIPCS(cmd)
	if err != nil {
		return nil, cli.Errorf(cli.CodeConfig, "failed to get base URI for PCS: %w", err)
	}

	// Create client to make request to PCS
	pcsClient, err := pcs.NewClient(pcsBaseURI, client.WithInsecure(cli.Insecure), client.WithShowToken(cli.ShowToken(cmd)))
	if err != nil {
		return nil, cli.Errorf(cli.CodeGeneric, "error creating new PCS client: %w", err)
	}

	// Check if a CA certificate was passed and load it into client if valid
	if err := cli.UseCACert(pcsClient.OchamiClient); err != nil {
		return nil, err
	}

	return pcsClient, nil
}
