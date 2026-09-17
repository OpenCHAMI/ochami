// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package smd

import (
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/smd"
	"github.com/openchami/ochami/pkg/config"
)

// GetClientWithRuntime sets up the SMD client with the SMD base URI and certificates
// (if necessary) and returns it. This function uses the provided runtime for configuration.
func GetClientWithRuntime(cmd *cobra.Command, rt *cli.Runtime) (*smd.SMDClient, error) {
	// Without a base URI, we cannot do anything
	smdBaseURI, err := rt.GetBaseURI(cmd, config.ServiceSMD)
	if err != nil {
		return nil, cli.Errorf(cli.CodeConfig, "failed to get base URI for SMD: %w", err)
	}

	// Create client to make request to SMD
	smdClient, err := smd.NewClient(smdBaseURI, client.WithInsecure(rt.Insecure), client.WithShowToken(rt.ShowToken(cmd)))
	if err != nil {
		return nil, cli.Errorf(cli.CodeGeneric, "error creating new SMD client: %w", err)
	}

	// Check if a CA certificate was passed and load it into client if valid
	if err := rt.UseCACert(smdClient.OchamiClient); err != nil {
		return nil, err
	}

	return smdClient, nil
}
