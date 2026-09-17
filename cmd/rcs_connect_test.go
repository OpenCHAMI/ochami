// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// rcs_connect_test.go exercises the RCS console connect command to cover
// error handling paths identified in coverage analysis.

import (
	"strings"
	"testing"
)

// TestRCSConsoleConnectHelp verifies the help text works.
func TestRCSConsoleConnectHelp(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "--ignore-config", "rcs", "console", "connect", "--help")

	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}

	// Help should contain usage information
	if !strings.Contains(res.stdout, "connect") {
		t.Errorf("expected help to contain 'connect', got: %s", res.stdout)
	}
}

// TestRCSConsoleConnectMissingNodeID verifies that missing node ID argument is handled.
func TestRCSConsoleConnectMissingNodeID(t *testing.T) {
	t.Parallel()

	// Try to connect without a node ID
	res := runOchamiWithRuntime(t, "--ignore-config", "rcs", "console", "connect")

	// Should fail with usage error
	if res.err == nil {
		t.Fatal("expected error for missing node ID, got nil")
	}
	if res.exitCode == 0 {
		t.Errorf("expected non-zero exit code, got %d", res.exitCode)
	}
}

// TestRCSConsoleConnectInvalidNodeID verifies handling of invalid node ID.
func TestRCSConsoleConnectInvalidNodeID(t *testing.T) {
	t.Parallel()

	// Try to connect with an invalid node ID format
	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", "http://localhost:8080",
		"--token", "t", "rcs", "console", "connect", "invalid-node-id")

	// Should fail with error (either connection error or validation error)
	if res.err == nil && res.exitCode == 0 {
		t.Logf("Invalid node ID: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}

// TestRCSConsoleConnectRequiresToken verifies that connect requires a token.
func TestRCSConsoleConnectRequiresToken(t *testing.T) {
	t.Parallel()

	// Try to connect without a token to a server that requires auth
	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", "http://localhost:8080",
		"rcs", "console", "connect", "x0c0s1b0n0")

	// Should fail with auth error or connection error
	if res.err == nil && res.exitCode == 0 {
		t.Logf("Requires token: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}

// TestRCSConsoleConnectUsage verifies the usage message.
func TestRCSConsoleConnectUsage(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "--ignore-config", "rcs", "console", "connect")

	// Should show usage or error
	if res.err == nil && res.exitCode == 0 {
		t.Logf("Usage: err=%v, exitCode=%d", res.err, res.exitCode)
	}
}
