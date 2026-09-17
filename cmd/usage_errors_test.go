// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// usage_errors_test.go verifies that usage errors across the command tree
// (unknown flags, bad flag values, and argument-count violations from Cobra's
// built-in Args validators) resolve to the CodeUsage exit code via the
// centralized WrapUsageErrors wiring in NewRootCmd.

import (
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestUsageErrorUnknownFlag verifies that an unknown flag is a usage error.
func TestUsageErrorUnknownFlag(t *testing.T) {
	res := runOchami(t, "smd", "component", "get", "--ignore-config", "--definitely-not-a-flag")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}

// TestUsageErrorBadFlagValue verifies that an invalid value for a typed flag
// (here, a non-integer for the int32 --nid) is a usage error.
func TestUsageErrorBadFlagValue(t *testing.T) {
	res := runOchami(t, "smd", "component", "get", "--ignore-config", "--nid", "not-a-number")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}

// TestUsageErrorTooManyArgs verifies that violating a command's Args validator
// (cobra.NoArgs on "smd component get") is a usage error.
func TestUsageErrorTooManyArgs(t *testing.T) {
	res := runOchami(t, "smd", "component", "get", "--ignore-config", "unexpected-arg")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}

// TestUsageErrorExactArgs verifies that an argument-count violation from
// cobra.ExactArgs (here, "smd group member get" requires exactly 1) is a usage
// error.
func TestUsageErrorExactArgs(t *testing.T) {
	res := runOchami(t, "smd", "group", "member", "get", "--ignore-config")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (CodeUsage)", res.exitCode, cli.CodeUsage)
	}
}
