// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package pcs

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client/pcs"
)

// GetClient sets up the PCS client with the PCS base URI and certificates
// (if necessary) and returns it. This function is used by each subcommand.
func GetClient(cmd *cobra.Command) *pcs.PCSClient {
	// Without a base URI, we cannot do anything
	pcsBaseURI, err := cli.GetBaseURIPCS(cmd)
	if err != nil {
		log.Logger.Error().Err(err).Msg("failed to get base URI for PCS")
		cli.LogHelpError(cmd)
		os.Exit(1)
	}

	// Create client to make request to PCS
	pcsClient, err := pcs.NewClient(pcsBaseURI, cli.Insecure)
	if err != nil {
		log.Logger.Fatal().Err(err).Msg("error creating new PCS client")
	}

	// Check if a CA certificate was passed and load it into client if valid
	cli.UseCACert(pcsClient.OchamiClient)

	return pcsClient
}
