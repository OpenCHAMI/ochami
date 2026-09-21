// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"testing"

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
