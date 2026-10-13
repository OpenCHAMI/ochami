// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// pcs_flags_test.go verifies that pcs status list's flag-backed variables are
// local to each command instance, mirroring the discover static flag-isolation
// test. The required --xname check for "pcs transition start" is in
// pcs_flags_errors_test.go.

import (
	"testing"

	pcs_status "github.com/openchami/ochami/cmd/pcs/status"
)

// TestPCSStatusList_FlagStateIsLocal verifies one "pcs status list" invocation
// cannot change the default --power-filter of a subsequently constructed
// command (xnames, powerFilter, and mgmtFilter are command-local variables,
// not package-level ones, for this reason).
func TestPCSStatusList_FlagStateIsLocal(t *testing.T) {
	t.Parallel()

	first := pcs_status.NewCmd()
	firstList, _, err := first.Find([]string{"list"})
	if err != nil {
		t.Fatalf("find first list command: %v", err)
	}
	if err := firstList.Flags().Set("power-filter", "on"); err != nil {
		t.Fatalf("set first power-filter: %v", err)
	}

	second := pcs_status.NewCmd()
	secondList, _, err := second.Find([]string{"list"})
	if err != nil {
		t.Fatalf("find second list command: %v", err)
	}
	if got := secondList.Flags().Lookup("power-filter").Value.String(); got != "" {
		t.Errorf("second command power-filter = %q, want default empty", got)
	}
}
