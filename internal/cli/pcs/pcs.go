// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package pcs

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/pcs"
	"github.com/openchami/ochami/pkg/config"
)

// GetClientWithRuntime sets up the PCS client with the PCS base URI and certificates
// (if necessary) and returns it. This function uses the provided runtime for configuration.
func GetClientWithRuntime(cmd *cobra.Command, rt *cli.Runtime) (*pcs.PCSClient, error) {
	// Without a base URI, we cannot do anything
	pcsBaseURI, err := rt.GetBaseURI(cmd, config.ServicePCS)
	if err != nil {
		return nil, cli.Errorf(cli.CodeConfig, "failed to get base URI for PCS: %w", err)
	}

	// Create client to make request to PCS
	pcsClient, err := pcs.NewClient(
		pcsBaseURI,
		client.WithInsecure(rt.Insecure),
		client.WithShowToken(rt.ShowToken(cmd)),
	)
	if err != nil {
		return nil, cli.Errorf(cli.CodeGeneric, "error creating new PCS client: %w", err)
	}

	// Check if a CA certificate was passed and load it into client if valid
	if err := rt.UseCACert(pcsClient.OchamiClient); err != nil {
		return nil, err
	}

	return pcsClient, nil
}

// GetClient sets up the PCS client with the PCS base URI and certificates
// (if necessary) and returns it. This function is used by each subcommand.
// During the transition to runtime, this function attempts to use runtime from context
// and falls back to global state if runtime is not available.
func GetClient(cmd *cobra.Command) (*pcs.PCSClient, error) {
	// Try to get runtime from context first (new approach)
	if rt, ok := cli.FromContext(cmd.Context()); ok {
		return GetClientWithRuntime(cmd, rt)
	}

	// Fallback to global state (old approach) during transition
	pcsBaseURI, err := cli.GetBaseURIPCS(cmd)
	if err != nil {
		return nil, cli.Errorf(cli.CodeConfig, "failed to get base URI for PCS: %w", err)
	}

	// Create client to make request to PCS
	pcsClient, err := pcs.NewClient(pcsBaseURI, client.WithInsecure(cli.Insecure), client.WithShowToken(cli.ShowToken(cmd)))
	if err != nil {
		return nil, cli.Errorf(cli.CodeGeneric, "error creating new PCS client: %w", err)
	}

	// Check if a CA certificate was passed and load it into client if valid
	if err := cli.UseCACert(pcsClient.OchamiClient); err != nil {
		return nil, err
	}

	return pcsClient, nil
}
