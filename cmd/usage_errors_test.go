// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// usage_errors_test.go verifies that usage errors across the command tree
// (unknown flags, bad flag values, and argument-count violations from Cobra's
// built-in Args validators) resolve to the CodeUsage exit code via the
// centralized WrapUsageErrors wiring in NewRootCmd.

import (
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestUsageError_UnknownFlag verifies that an unknown flag is a usage error.
func TestUsageError_UnknownFlag(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "--ignore-config", "smd", "component", "get", "--definitely-not-a-flag")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestUsageError_BadFlagValue verifies that an invalid value for a typed flag
// (here, a non-integer for the int32 --nid) is a usage error.
func TestUsageError_BadFlagValue(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "--ignore-config", "smd", "component", "get", "--nid", "not-a-number")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestUsageError_TooManyArgs verifies that violating a command's Args validator
// (cobra.NoArgs on "smd component get") is a usage error.
func TestUsageError_TooManyArgs(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "--ignore-config", "smd", "component", "get", "unexpected-arg")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestUsageError_ExactArgs verifies that an argument-count violation from
// cobra.ExactArgs (here, "smd group member get" requires exactly 1) is a usage
// error.
func TestUsageError_ExactArgs(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "--ignore-config", "smd", "group", "member", "get")
	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}

// TestUsageError_UnknownCommand verifies that an unknown subcommand of the
// root command is a usage error.
func TestUsageError_UnknownCommand(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "invalid-command")
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s): %v", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage), res.err)
	}
}

// TestUsageError_PatchFormatInputWithKeyValFlags verifies that the patch
// commands reject --format-input combined with a key-value flag (--set,
// --unset, --add, or --remove), which builds the patch without reading a
// payload.
func TestUsageError_PatchFormatInputWithKeyValFlags(t *testing.T) {
	t.Parallel()

	for _, resource := range []string{"boot bmc", "boot config", "boot node", "metadata defaults", "metadata group", "metadata instance", "metadata peer"} {
		args := append(strings.Fields(resource), "patch", "uid", "--set", "a=b", "-f", "yaml",
			"--ignore-config", "--uri", "http://127.0.0.1:1", "--no-token")
		res := runOchamiWithRuntime(t, args...)
		if res.exitCode != cli.CodeUsage {
			t.Errorf("%s patch: result = (err %v, exit %d), want %d (%s)", resource, res.err, res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
		}
	}
}
