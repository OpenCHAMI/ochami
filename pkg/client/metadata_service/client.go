// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package metadata_service

import (
	"fmt"
	"time"

	metadata_service_client "github.com/openchami/metadata-service/pkg/client"
	"github.com/rs/zerolog"

	"github.com/openchami/ochami/pkg/client"
)

const (
	serviceNameMetadataService = "metadata-service"
)

// MetadataServiceClient is an OchamiClient that has its BasePath set configured to
// the one that metadata-service uses.
type MetadataServiceClient struct {
	*client.OchamiClient
	Client  *metadata_service_client.Client
	Timeout time.Duration
}

// NewClient takes a baseURI, timeout duration, optional API version string, and
// logger and returns a pointer to a new MetadataServiceClient.  If an error
// occurred creating the embedded OchamiClient or the metadata service client,
// it is returned. Behavior such as TLS verification and token redaction is
// configured via functional options (e.g. client.WithInsecure,
// client.WithShowToken). An API version can also be specified (e.g. 'v1beta2'),
// though it can be left blank to use the default.
func NewClient(baseURI string, timeout time.Duration, apiVersion string, logger zerolog.Logger, opts ...client.Option) (*MetadataServiceClient, error) {
	// Create OchamiClient to ensure http client is configured via ochami CLI
	// flags/config.
	oc, err := client.NewOchamiClient(serviceNameMetadataService, baseURI, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OchamiClient for %s: %w", serviceNameMetadataService, err)
	}

	// Create metadata-service client via its API, using the http client from
	// the OchamiClient so that passed certs or --insecure is honored.
	msc, err := metadata_service_client.NewClient(baseURI, oc.Client, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s client: %w", serviceNameMetadataService, err)
	}
	msc = msc.WithShowToken(oc.ShowToken)

	// Optionally set API version, if passed.
	if apiVersion != "" {
		msc = msc.WithVersion(apiVersion)
	}

	// Aggregate the clients into one struct.
	mc := &MetadataServiceClient{
		OchamiClient: oc,
		Client:       msc,
		Timeout:      timeout,
	}

	return mc, err
}
