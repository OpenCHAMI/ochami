// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package boot_service

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/boot_service"
	"github.com/openchami/ochami/pkg/config"
)

// GetClientWithRuntime sets up the boot-service client with the boot-service base URI and
// certificates (if necessary) and returns it. This function uses the provided runtime for configuration.
func GetClientWithRuntime(cmd *cobra.Command, rt *cli.Runtime) (*boot_service.BootServiceClient, error) {
	// Without a base URI, we cannot do anything
	bootServiceBaseURI, err := rt.GetBaseURI(cmd, config.ServiceBoot)
	if err != nil {
		return nil, cli.Errorf(cli.CodeConfig, "failed to get base URI for boot-service: %w", err)
	}

	apiVersion, err := rt.GetAPIVersion(cmd, config.ServiceBoot)
	if err != nil {
		rt.Logger.Warn().Err(err).Msgf("failed to determine API version for %s from user, skipping", config.ServiceBoot)
	}

	// Create client to make request to boot-service
	bootServiceClient, err := boot_service.NewClient(bootServiceBaseURI, rt.GetTimeout(cmd), apiVersion, rt.Logger, client.WithInsecure(rt.Insecure), client.WithShowToken(rt.ShowToken(cmd)), client.WithLogger(rt.Logger))
	if err != nil {
		return nil, cli.Errorf(cli.CodeGeneric, "error creating new boot-service client: %w", err)
	}

	// Check if a CA certificate was passed and load it into client if valid
	if err := rt.UseCACert(bootServiceClient.OchamiClient); err != nil {
		return nil, err
	}

	return bootServiceClient, nil
}

// GetClient sets up the boot-service client with the boot-service base URI and
// certificates (if necessary) and returns it. This function uses the runtime from context.
// Since cmd/root.go always injects a runtime into context, this will always succeed.
func GetClient(cmd *cobra.Command) (*boot_service.BootServiceClient, error) {
	// Get runtime from context (always available since cmd/root.go injects it)
	rt, err := cli.RuntimeFromCommand(cmd)
	if err != nil {
		return nil, err
	}
	return GetClientWithRuntime(cmd, rt)
}
