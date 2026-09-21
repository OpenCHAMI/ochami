// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

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
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}
