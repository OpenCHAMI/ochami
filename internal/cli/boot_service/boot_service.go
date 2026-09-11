// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package boot_service

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/internal/log"
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
		log.Logger.Warn().Err(err).Msgf("failed to determine API version for %s from user, skipping", config.ServiceBoot)
	}

	// Create client to make request to boot-service
	bootServiceClient, err := boot_service.NewClient(bootServiceBaseURI, rt.GetTimeout(cmd), apiVersion, log.Logger, client.WithInsecure(rt.Insecure), client.WithShowToken(rt.ShowToken(cmd)))
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
// certificates (if necessary) and returns it. This function is used by each
// subcommand.
// During the transition to runtime, this function attempts to use runtime from context
// and falls back to global state if runtime is not available.
func GetClient(cmd *cobra.Command) (*boot_service.BootServiceClient, error) {
	// Try to get runtime from context first (new approach)
	if rt, ok := cli.FromContext(cmd.Context()); ok {
		return GetClientWithRuntime(cmd, rt)
	}

	// Fallback to global state (old approach) during transition
	// Without a base URI, we cannot do anything
	bootServiceBaseURI, err := cli.GetBaseURIBootService(cmd)
	if err != nil {
		return nil, cli.Errorf(cli.CodeConfig, "failed to get base URI for boot-service: %w", err)
	}

	apiVersion, err := cli.GetAPIVersion(cmd, config.ServiceBoot)
	if err != nil {
		log.Logger.Warn().Err(err).Msgf("failed to determine API version for %s from user, skipping", config.ServiceBoot)
	}

	// Create client to make request to boot-service
	bootServiceClient, err := boot_service.NewClient(bootServiceBaseURI, cli.GetTimeout(cmd), apiVersion, log.Logger, client.WithInsecure(cli.Insecure), client.WithShowToken(cli.ShowToken(cmd)))
	if err != nil {
		return nil, cli.Errorf(cli.CodeGeneric, "error creating new boot-service client: %w", err)
	}

	// Check if a CA certificate was passed and load it into client if valid
	if err := cli.UseCACert(bootServiceClient.OchamiClient); err != nil {
		return nil, err
	}

	return bootServiceClient, nil
}
