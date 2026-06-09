// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/config"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client/metadata_service"
)

// GetClient sets up the metadata-service client with the metadata-service base
// URI and certificates (if necessary) and returns it. This function is used by
// each subcommand.
func GetClient(cmd *cobra.Command) *metadata_service.MetadataServiceClient {
	// Without a base URI, we cannot do anything
	metadataServiceBaseURI, err := cli.GetBaseURIMetadataService(cmd)
	if err != nil {
		log.Logger.Error().Err(err).Msg("failed to get base URI for metadata-service")
		cli.LogHelpError(cmd)
		os.Exit(1)
	}

	apiVersion, err := cli.GetAPIVersion(cmd, config.ServiceMetadata)
	if err != nil {
		log.Logger.Warn().Err(err).Msgf("failed to determine API version for %s from user, skipping", config.ServiceMetadata)
	}

	// Create client to make request to metadata-service
	metadataServiceClient, err := metadata_service.NewClient(metadataServiceBaseURI, cli.Insecure, cli.GetTimeout(cmd), apiVersion, log.Logger)
	if err != nil {
		log.Logger.Error().Err(err).Msg("error creating new metadata-service client")
		cli.LogHelpError(cmd)
		os.Exit(1)
	}

	// Check if a CA certificate was passed and load it into client if valid
	cli.UseCACert(metadataServiceClient.OchamiClient)

	return metadataServiceClient
}
