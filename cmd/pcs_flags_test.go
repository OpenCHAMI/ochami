// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// pcs_flags_test.go verifies that pcs status/transition's flag-backed
// variables are local to each command instance (mirroring the discover
// static flag-isolation test), and that "pcs transition start" enforces its
// required --xname flag.

import (
	"testing"

	pcs_status "github.com/openchami/ochami/cmd/pcs/status"

	"github.com/openchami/ochami/internal/cli"
)

// TestPCSStatusList_FlagStateIsLocal verifies one "pcs status list" invocation
// cannot change the default --power-filter of a subsequently constructed
// command (pollInterval/xnames/powerFilter/mgmtFilter were moved from
// package-level vars to command-local ones for the same reason).
func TestPCSStatusList_FlagStateIsLocal(t *testing.T) {
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

// TestPCSTransitionStart_RequiresXname verifies --xname is enforced as a
// required flag (proving the MarkFlagRequired registration in start.go takes
// effect; the registration's own error return is unreachable in practice
// since "xname" is always registered immediately beforehand), and that
// Cobra's required-flag validation error resolves to CodeUsage via
// cli.WrapUsageErrors, which composes each command's PreRunE/PreRun to
// re-run and wrap Cobra's ValidateRequiredFlags/ValidateFlagGroups checks.
func TestPCSTransitionStart_RequiresXname(t *testing.T) {
	res := runOchamiWithRuntime(t, "pcs", "transition", "start", "--ignore-config",
		"--uri", "http://127.0.0.1:1", "--token", "t", "on")
	if res.err == nil {
		t.Fatal("expected an error for missing required --xname, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}
