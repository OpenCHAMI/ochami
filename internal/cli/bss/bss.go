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
	"github.com/openchami/ochami/pkg/config"
)

// GetClientWithRuntime sets up the BSS client with the BSS base URI and certificates
// (if necessary) and returns it. This function uses the provided runtime for configuration.
func GetClientWithRuntime(cmd *cobra.Command, rt *cli.Runtime) (*bss.BSSClient, error) {
	// Without a base URI, we cannot do anything
	bssBaseURI, err := rt.GetBaseURI(cmd, config.ServiceBSS)
	if err != nil {
		return nil, cli.Errorf(cli.CodeConfig, "failed to get base URI for BSS: %w", err)
	}

	// Create client to make request to BSS
	bssClient, err := bss.NewClient(bssBaseURI, client.WithInsecure(rt.Insecure), client.WithShowToken(rt.ShowToken(cmd)), client.WithLogger(rt.Logger))
	if err != nil {
		return nil, cli.Errorf(cli.CodeGeneric, "error creating new BSS client: %w", err)
	}

	// Check if a CA certificate was passed and load it into client if valid
	if err := rt.UseCACert(bssClient.OchamiClient); err != nil {
		return nil, err
	}

	return bssClient, nil
}
