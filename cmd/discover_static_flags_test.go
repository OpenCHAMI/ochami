// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// discover_static_flags_test.go verifies that discover static's flag-backed
// variables are local to each command instance and do not leak state across
// repeated NewCmd() construction.

import (
	"testing"

	discover_static "github.com/openchami/ochami/cmd/discover/static"
)

// TestDiscoverStatic_FlagStateIsLocal verifies one command invocation cannot
// change the default discovery version of a subsequently constructed command.
func TestDiscoverStatic_FlagStateIsLocal(t *testing.T) {
	first := discover_static.NewCmd()
	if err := first.Flags().Set("discovery-version", "1"); err != nil {
		t.Fatalf("set first discovery version: %v", err)
	}

	second := discover_static.NewCmd()
	if got := second.Flags().Lookup("discovery-version").Value.String(); got != "2" {
		t.Errorf("second command discovery version = %q, want default 2", got)
	}
}
