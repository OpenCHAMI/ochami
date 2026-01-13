// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package smd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client/smd"
)

// GetClient sets up the SMD client with the SMD base URI and certificates
// (if necessary) and returns it. This function is used by each subcommand.
func GetClient(cmd *cobra.Command) *smd.SMDClient {
	// Without a base URI, we cannot do anything
	smdBaseURI, err := cli.GetBaseURISMD(cmd)
	if err != nil {
		log.Logger.Error().Err(err).Msg("failed to get base URI for SMD")
		cli.LogHelpError(cmd)
		os.Exit(1)
	}

	// Create client to make request to SMD
	smdClient, err := smd.NewClient(smdBaseURI, cli.Insecure)
	if err != nil {
		log.Logger.Error().Err(err).Msg("error creating new SMD client")
		cli.LogHelpError(cmd)
		os.Exit(1)
	}

	// Check if a CA certificate was passed and load it into client if valid
	cli.UseCACert(smdClient.OchamiClient)

	return smdClient
}
