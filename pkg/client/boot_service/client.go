// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package boot_service

import (
	"fmt"
	"time"

	boot_service_client "github.com/openchami/boot-service/pkg/client"
	"github.com/rs/zerolog"

	"github.com/openchami/ochami/pkg/client"
)

const (
	serviceNameBootService = "boot-service"
)

// BootServiceClient is an OchamiClient that has its BasePath set configured to
// the one that boot-service uses.
type BootServiceClient struct {
	*client.OchamiClient
	Client  *boot_service_client.Client
	Timeout time.Duration
}

// NewClient takes a baseURI, timeout duration, optional API version string, and
// logger and returns a pointer to a new BootServiceClient.  If an error
// occurred creating the embedded OchamiClient or the boot service client, it is
// returned. Behavior such as TLS verification and token redaction is configured
// via functional options (e.g. client.WithInsecure, client.WithShowToken). An
// API version can also be specified (e.g. 'v1beta2'), though it can be left
// blank to use the default.
func NewClient(baseURI string, timeout time.Duration, apiVersion string, logger zerolog.Logger, opts ...client.Option) (*BootServiceClient, error) {
	// Create OchamiClient to ensure http client is configured via ochami CLI
	// flags/config.
	oc, err := client.NewOchamiClient(serviceNameBootService, baseURI, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OchamiClient for %s: %w", serviceNameBootService, err)
	}

	// Create boot-service client via its API, using the http client from the
	// OchamiClient so that passed certs or --insecure is honored.
	bsc, err := boot_service_client.NewClient(baseURI, oc.Client, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s client: %w", serviceNameBootService, err)
	}
	bsc = bsc.WithShowToken(oc.ShowToken)

	// Optionally set API version, if passed.
	if apiVersion != "" {
		bsc = bsc.WithVersion(apiVersion)
	}

	// Aggregate the clients into one struct.
	bc := &BootServiceClient{
		OchamiClient: oc,
		Client:       bsc,
		Timeout:      timeout,
	}

	return bc, err
}
