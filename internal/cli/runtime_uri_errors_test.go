// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/config"
)

// TestGetBaseURI_UnknownServiceWithURIFlag verifies that GetBaseURI rejects an
// unknown service even when --uri is set.
func TestGetBaseURI_UnknownServiceWithURIFlag(t *testing.T) {
	rt := NewTestRuntime(nil, nil, nil)
	cmd := newURICmd()
	if err := cmd.Flags().Set("uri", "https://x.example.com"); err != nil {
		t.Fatalf("set uri flag: %v", err)
	}

	if _, err := rt.GetBaseURI(cmd, config.ServiceName("bogus")); err == nil {
		t.Fatal("expected error for unknown service with --uri, got nil")
	}
}

// TestGetBaseURI_IncludesClusterContextInError verifies that when the default
// cluster has no URI for a service, GetBaseURI's error names the cluster.
func TestGetBaseURI_IncludesClusterContextInError(t *testing.T) {
	rt := NewRuntime().WithConfig(config.Config{
		DefaultCluster: "demo",
		Clusters:       []config.Cluster{{Name: "demo"}},
	})
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("cluster", "", "")
	cmd.Flags().String("cluster-uri", "", "")
	cmd.Flags().String("uri", "", "")
	_, err := rt.GetBaseURI(cmd, config.ServiceSMD)
	if err == nil || !strings.Contains(err.Error(), "cluster demo") {
		t.Fatalf("GetBaseURI() error = %v, want cluster context", err)
	}
}
