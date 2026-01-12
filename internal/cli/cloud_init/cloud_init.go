// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client/cloud_init"
)

type CIFlagHeaderWhen string

const (
	CIFlagHeaderAlways   = "always"
	CIFlagHeaderMultiple = "multiple"
	CIFlagHeaderNever    = "never"
)

var (
	CIFlagHeaderWhenHelp = map[string]string{
		string(CIFlagHeaderAlways):   "Always print headers, even if singular output",
		string(CIFlagHeaderMultiple): "Only print headers if multiple items in output",
		string(CIFlagHeaderNever):    "Never print headers",
	}
	CIHeaderWhen CIFlagHeaderWhen = CIFlagHeaderMultiple
)

func (cfhw CIFlagHeaderWhen) String() string {
	return string(cfhw)
}

func (cfhw *CIFlagHeaderWhen) Set(v string) error {
	switch CIFlagHeaderWhen(v) {
	case CIFlagHeaderAlways,
		CIFlagHeaderMultiple,
		CIFlagHeaderNever:
		*cfhw = CIFlagHeaderWhen(v)
		return nil
	default:
		return fmt.Errorf("must be one of %v", []CIFlagHeaderWhen{
			CIFlagHeaderAlways,
			CIFlagHeaderMultiple,
			CIFlagHeaderNever,
		})
	}
}

func (cfhw CIFlagHeaderWhen) Type() string {
	return "CIFlagHeaderWhen"
}

func CompletionHeaderWhen(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var helpSlice []string
	for k, v := range CIFlagHeaderWhenHelp {
		helpSlice = append(helpSlice, fmt.Sprintf("%s\t%s", k, v))
	}
	return helpSlice, cobra.ShellCompDirectiveDefault
}

// GetClient sets up the cloud-init client with the cloud-init base URI
// and certificates (if necessary) and returns it. This function is used by
// each subcommand.
func GetClient(cmd *cobra.Command) *cloud_init.CloudInitClient {
	// Without a base URI, we cannot do anything
	cloudInitbaseURI, err := cli.GetBaseURICloudInit(cmd)
	if err != nil {
		log.Logger.Error().Err(err).Msg("failed to get base URI for cloud-init")
		cli.LogHelpError(cmd)
		os.Exit(1)
	}

	// Create client to make request to cloud-init
	cloudInitClient, err := cloud_init.NewClient(cloudInitbaseURI, cli.Insecure)
	if err != nil {
		log.Logger.Error().Err(err).Msg("error creating new cloud-init client")
		cli.LogHelpError(cmd)
		os.Exit(1)
	}

	// Check if a CA certificate was passed and load it into client if valid
	cli.UseCACert(cloudInitClient.OchamiClient)

	return cloudInitClient
}
