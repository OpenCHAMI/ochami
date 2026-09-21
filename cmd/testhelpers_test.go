// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// testhelpers_test.go provides shared helpers for command-level (end-to-end)
// tests. These tests build the real root command tree, point commands at an
// httptest.Server via the global --uri flag, and assert both the outbound
// request shape and the resolved process exit code (via cli.ExitCode) that the
// command's returned error maps to. Config file reading is disabled with
// --ignore-config so tests never touch a user's real configuration.
//
// All test helpers now use runtime-based execution for better test isolation
// and to enable parallel test execution.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

func writeJSONResponse(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Errorf("encode test response: %v", err)
	}
}

// cmdResult captures everything a command-level test needs to assert on after
// running the CLI: the error returned from Execute, the exit code that error
// resolves to, and whatever the command wrote to the combined stdout/stderr buffer.
type cmdResult struct {
	err      error
	exitCode int
	stdout   string
}

// runOchamiWithRuntime executes the ochami root command with an isolated Runtime,
// enabling test isolation and parallel test execution. Each test gets its own Runtime
// instance with isolated I/O streams, eliminating the need for global state manipulation.
//
// Callers should generally include "--ignore-config" so the command does not
// read or create real config files, and "--uri <server.URL>" (on commands that
// accept it) to target an httptest.Server.
func runOchamiWithRuntime(t *testing.T, args ...string) cmdResult {
	t.Helper()
	return runOchamiWithRuntimeEnv(t, nil, args...)
}

// runOchamiWithRuntimeEnv is like runOchamiWithRuntime, but installs env as
// the runtime's Environment when non-nil. NewTestRuntime's environment is
// hermetic by default, so tests that rely on real process environment
// variables (e.g. via t.Setenv) must opt in explicitly through this helper.
func runOchamiWithRuntimeEnv(t *testing.T, env cli.Environment, args ...string) cmdResult {
	t.Helper()

	// Create isolated runtime for this test with custom I/O streams
	// Use a single buffer for both stdout and stderr to capture all command output.
	// This is important for tests that check for interactive prompts (written to stderr
	// via rt.Ios.LoopYesNo) in the stdout.
	var combinedBuf bytes.Buffer
	stdinReader := strings.NewReader("")

	// Create runtime with both stdout and stderr pointing to the same buffer
	rt := cli.NewTestRuntime(stdinReader, &combinedBuf, &combinedBuf)
	if env != nil {
		rt = rt.WithEnvironment(env)
	}

	// Create root command with runtime in context
	rootCmd := NewRootCmd()
	rootCmd.SetContext(cli.ContextWithRuntime(context.Background(), rt))
	rootCmd.SetArgs(args)
	// Set both Cobra's output streams to the same buffer for consistency
	rootCmd.SetOut(&combinedBuf)
	rootCmd.SetErr(&combinedBuf)

	// Execute command - runtime is automatically used via context
	runErr := rootCmd.Execute()

	return cmdResult{
		err:      runErr,
		exitCode: cli.ExitCode(runErr),
		stdout:   combinedBuf.String(),
	}
}

// runOchamiWithInputAndRuntime executes the ochami root command with an isolated
// Runtime and custom stdin input, enabling test isolation and parallel test execution.
func runOchamiWithInputAndRuntime(t *testing.T, input string, args ...string) cmdResult {
	t.Helper()

	// Create isolated runtime for this test with custom I/O streams
	// Use a single buffer for both stdout and stderr to capture all command output.
	// This is important for tests that check for interactive prompts (written to stderr
	// via rt.Ios.LoopYesNo) in the stdout.
	var combinedBuf bytes.Buffer
	stdinReader := strings.NewReader(input)

	// Create runtime with both stdout and stderr pointing to the same buffer
	rt := cli.NewTestRuntime(stdinReader, &combinedBuf, &combinedBuf)

	// Create root command with runtime in context
	rootCmd := NewRootCmd()
	rootCmd.SetContext(cli.ContextWithRuntime(context.Background(), rt))
	rootCmd.SetArgs(args)
	// Set both Cobra's output streams to the same buffer for consistency
	rootCmd.SetOut(&combinedBuf)
	rootCmd.SetErr(&combinedBuf)

	// Execute command
	runErr := rootCmd.Execute()

	return cmdResult{
		err:      runErr,
		exitCode: cli.ExitCode(runErr),
		stdout:   combinedBuf.String(),
	}
}

// TestRunOchamiWithRuntime_Basic verifies that the runtime-based test helper
// works correctly for basic command execution.
func TestRunOchamiWithRuntime_Basic(t *testing.T) {
	t.Parallel()

	// Test version command which should work without any special setup
	res := runOchamiWithRuntime(t, "--ignore-config", "version")

	// Version command should succeed
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}

	// Should have some output
	if res.stdout == "" {
		t.Error("expected version output, got empty string")
	}

	// Output should contain version information
	if !strings.Contains(res.stdout, "Version:") {
		t.Errorf("expected output to contain 'Version:', got: %s", res.stdout)
	}
}

// TestRunOchamiWithInputAndRuntime_Basic verifies that the runtime-based test
// helper with input works correctly.
func TestRunOchamiWithInputAndRuntime_Basic(t *testing.T) {
	t.Parallel()

	// Test version command with custom input (should be ignored by version)
	res := runOchamiWithInputAndRuntime(t, "some input", "--ignore-config", "version")

	// Version command should succeed
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}

	// Should have some output
	if res.stdout == "" {
		t.Error("expected version output, got empty string")
	}

	// Output should contain version information
	if !strings.Contains(res.stdout, "Version:") {
		t.Errorf("expected output to contain 'Version:', got: %s", res.stdout)
	}
}
