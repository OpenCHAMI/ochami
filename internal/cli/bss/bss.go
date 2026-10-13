// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package bss

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/bss"
)

// GetClient sets up the BSS client with the BSS base URI and certificates
// (if necessary) and returns it. This function is used by each subcommand.
func GetClient(cmd *cobra.Command) (*bss.BSSClient, error) {
	// Without a base URI, we cannot do anything
	bssBaseURI, err := cli.GetBaseURIBSS(cmd)
	if err != nil {
		return nil, cli.Errorf(cli.CodeConfig, "failed to get base URI for BSS: %w", err)
	}

	// Create client to make request to BSS
	bssClient, err := bss.NewClient(bssBaseURI, client.WithInsecure(cli.Insecure), client.WithShowToken(cli.ShowToken(cmd)))
	if err != nil {
		return nil, cli.Errorf(cli.CodeGeneric, "error creating new BSS client: %w", err)
	}

	// Check if a CA certificate was passed and load it into client if valid
	if err := cli.UseCACert(bssClient.OchamiClient); err != nil {
		return nil, err
	}

	return bssClient, nil
}
