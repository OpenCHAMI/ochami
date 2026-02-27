// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package boot_service

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/config"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client/boot_service"
)

// GetClient sets up the BSS client with the BSS base URI and certificates
// (if necessary) and returns it. This function is used by each subcommand.
func GetClient(cmd *cobra.Command) *boot_service.BootServiceClient {
	// Without a base URI, we cannot do anything
	bootServiceBaseURI, err := cli.GetBaseURIBootService(cmd)
	if err != nil {
		log.Logger.Error().Err(err).Msg("failed to get base URI for boot-service")
		cli.LogHelpError(cmd)
		os.Exit(1)
	}

	apiVersion, err := cli.GetAPIVersion(cmd, config.ServiceBoot)
	if err != nil {
		log.Logger.Warn().Err(err).Msgf("failed to determine API version for %s from user, skipping", config.ServiceBoot)
	}

	// Create client to make request to boot-service
	bootServiceClient, err := boot_service.NewClient(bootServiceBaseURI, cli.Insecure, cli.GetTimeout(cmd), apiVersion)
	if err != nil {
		log.Logger.Error().Err(err).Msg("error creating new boot-service client")
		cli.LogHelpError(cmd)
		os.Exit(1)
	}

	// Check if a CA certificate was passed and load it into client if valid
	cli.UseCACert(bootServiceClient.OchamiClient)

	return bootServiceClient
}
