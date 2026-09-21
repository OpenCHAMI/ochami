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
)

// GetClient sets up the SMD client with the SMD base URI and certificates
// (if necessary) and returns it. This function is used by each subcommand.
func GetClient(cmd *cobra.Command) (*smd.SMDClient, error) {
	// Without a base URI, we cannot do anything
	smdBaseURI, err := cli.GetBaseURISMD(cmd)
	if err != nil {
		return nil, cli.Errorf(cli.CodeConfig, "failed to get base URI for SMD: %w", err)
	}

	// Create client to make request to SMD
	smdClient, err := smd.NewClient(smdBaseURI, client.WithInsecure(cli.Insecure), client.WithShowToken(cli.ShowToken(cmd)))
	if err != nil {
		return nil, cli.Errorf(cli.CodeGeneric, "error creating new SMD client: %w", err)
	}

	// Check if a CA certificate was passed and load it into client if valid
	if err := cli.UseCACert(smdClient.OchamiClient); err != nil {
		return nil, err
	}

	return smdClient, nil
}

// CheckIDs returns a CodeUsage error for the first of ids that cannot name a
// single SMD resource (see client.CheckPathSegment), so that a delete command
// can reject its whole selection before sending any request. what names the
// IDs in the error message (e.g. "component ID"), which also gives the
// 1-based position of the rejected ID.
func CheckIDs(what string, ids []string) error {
	for i, id := range ids {
		if err := client.CheckPathSegment(what, id); err != nil {
			return cli.Errorf(cli.CodeUsage, "%w (item %d of %d)", err, i+1, len(ids))
		}
	}
	return nil
}
