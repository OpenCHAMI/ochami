// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package bss

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client/bss"
)

// GetClient sets up the BSS client with the BSS base URI and certificates
// (if necessary) and returns it. This function is used by each subcommand.
func GetClient(cmd *cobra.Command) *bss.BSSClient {
	// Without a base URI, we cannot do anything
	bssBaseURI, err := cli.GetBaseURIBSS(cmd)
	if err != nil {
		log.Logger.Error().Err(err).Msg("failed to get base URI for BSS")
		cli.LogHelpError(cmd)
		os.Exit(1)
	}

	// Create client to make request to BSS
	bssClient, err := bss.NewClient(bssBaseURI, cli.Insecure)
	if err != nil {
		log.Logger.Error().Err(err).Msg("error creating new BSS client")
		cli.LogHelpError(cmd)
		os.Exit(1)
	}

	// Check if a CA certificate was passed and load it into client if valid
	cli.UseCACert(bssClient.OchamiClient)

	return bssClient
}
