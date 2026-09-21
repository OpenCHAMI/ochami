// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/client/cloud_init"
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
)

// RenderItem is one cloud-init document and the labels used in its header.
type RenderItem struct {
	Labels                     string
	Body                       string
	BlankLineAfterAlwaysHeader bool
}

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

// Render writes cloud-init documents with headers according to headerWhen.
func Render(w io.Writer, headerWhen CIFlagHeaderWhen, items []RenderItem) error {
	showHeaders := headerWhen == CIFlagHeaderAlways ||
		(headerWhen == CIFlagHeaderMultiple && len(items) > 1)

	for idx, item := range items {
		if showHeaders {
			if err := writeString(w, fmt.Sprintf("--- (%d/%d) %s\n", idx+1, len(items), item.Labels)); err != nil {
				return err
			}
		}
		if err := writeString(w, item.Body+"\n"); err != nil {
			return err
		}
		if headerWhen == CIFlagHeaderAlways && item.BlankLineAfterAlwaysHeader {
			if err := writeString(w, "\n"); err != nil {
				return err
			}
		}
	}

	return nil
}

func writeString(w io.Writer, value string) error {
	n, err := io.WriteString(w, value)
	if err != nil {
		return err
	}
	if n != len(value) {
		return io.ErrShortWrite
	}
	return nil
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
func GetClient(cmd *cobra.Command) (*cloud_init.CloudInitClient, error) {
	// Without a base URI, we cannot do anything
	cloudInitbaseURI, err := cli.GetBaseURICloudInit(cmd)
	if err != nil {
		return nil, cli.Errorf(cli.CodeConfig, "failed to get base URI for cloud-init: %w", err)
	}

	// Create client to make request to cloud-init
	cloudInitClient, err := cloud_init.NewClient(cloudInitbaseURI, client.WithInsecure(cli.Insecure), client.WithShowToken(cli.ShowToken(cmd)))
	if err != nil {
		return nil, cli.Errorf(cli.CodeGeneric, "error creating new cloud-init client: %w", err)
	}

	// Check if a CA certificate was passed and load it into client if valid
	if err := cli.UseCACert(cloudInitClient.OchamiClient); err != nil {
		return nil, err
	}

	return cloudInitClient, nil
}
