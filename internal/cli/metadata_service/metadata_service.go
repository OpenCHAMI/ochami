// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/metadata_service"
	"github.com/openchami/ochami/pkg/config"
)

// GetClient sets up the metadata-service client with the metadata-service base
// URI and certificates (if necessary) and returns it. This function is used by
// each subcommand.
func GetClient(cmd *cobra.Command) (*metadata_service.MetadataServiceClient, error) {
	// Without a base URI, we cannot do anything
	metadataServiceBaseURI, err := cli.GetBaseURIMetadataService(cmd)
	if err != nil {
		return nil, cli.Errorf(cli.CodeConfig, "failed to get base URI for metadata-service: %w", err)
	}

	apiVersion, err := cli.GetAPIVersion(cmd, config.ServiceMetadata)
	if err != nil {
		log.Logger.Warn().Err(err).Msgf("failed to determine API version for %s from user, skipping", config.ServiceMetadata)
	}

	// Create client to make request to metadata-service
	metadataServiceClient, err := metadata_service.NewClient(metadataServiceBaseURI, cli.GetTimeout(cmd), apiVersion, log.Logger, client.WithInsecure(cli.Insecure), client.WithShowToken(cli.ShowToken(cmd)))
	if err != nil {
		return nil, cli.Errorf(cli.CodeGeneric, "error creating new metadata-service client: %w", err)
	}

	// Check if a CA certificate was passed and load it into client if valid
	if err := cli.UseCACert(metadataServiceClient.OchamiClient); err != nil {
		return nil, err
	}

	return metadataServiceClient, nil
}
