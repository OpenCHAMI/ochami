// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rcs

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client/rcs"
)

// GetClient sets up the remote-console client with the base URI and certificates
// (if necessary) and returns it. This function is used by each subcommand.
func GetClient(cmd *cobra.Command) *rcs.RCSClient {
	rcsBaseURI, err := cli.GetBaseURIRCS(cmd)
	if err != nil {
		log.Logger.Error().Err(err).Msg("failed to get base URI for remote-console")
		cli.LogHelpError(cmd)
		os.Exit(1)
	}

	insecure, _ := cmd.Flags().GetBool("insecure")

	rcsClient, err := rcs.NewClient(rcsBaseURI, insecure)
	if err != nil {
		log.Logger.Error().Err(err).Msg("error creating new remote-console client")
		cli.LogHelpError(cmd)
		os.Exit(1)
	}

	cli.UseCACert(rcsClient.OchamiClient)

	return rcsClient
}
