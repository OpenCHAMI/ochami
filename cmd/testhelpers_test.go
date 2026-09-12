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

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
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
// resolves to, and whatever the command wrote to os.Stdout.
type cmdResult struct {
	err      error
	exitCode int
	stdout   string
}

// stdoutMu serializes tests that capture os.Stdout. Because commands print
// directly to os.Stdout (via fmt.Print), and Go runs tests within a package
// sequentially by default but subtests/parallel tests could interleave, we
// guard the global swap with a mutex.
var stdoutMu sync.Mutex

// testStdin, when non-nil, is used as the interactive input stream for the next
// runOchami call. runOchamiWithInput sets it so commands that prompt (e.g.
// delete confirmations) read a scripted answer. It is reset after each run.
var testStdin io.Reader

// runOchami executes the ochami root command with the provided arguments,
// capturing anything written to os.Stdout during execution. It returns a
// cmdResult with the command error, the exit code cli.ExitCode maps that error
// to, and the captured stdout.
//
// Callers should generally include "--ignore-config" so the command does not
// read or create real config files, and "--uri <server.URL>" (on commands that
// accept it) to target an httptest.Server.
func runOchami(t *testing.T, args ...string) cmdResult {
	t.Helper()

	// Reset the token global between runs so a token set by a previous test
	// does not leak into this one.
	cli.Token = ""

	stdoutMu.Lock()
	defer stdoutMu.Unlock()

	// Redirect os.Stdout to a pipe so we can capture command output.
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	// Drain the pipe in a goroutine so a command writing more than the pipe
	// buffer does not deadlock.
	outCh := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r); err != nil {
			t.Errorf("copy command output: %v", err)
		}
		outCh <- buf.String()
	}()

	// Redirect the interactive I/O stream to the same capture pipe so output
	// written via cli.Ios.Out() (e.g. "rcs console show") and any interactive
	// prompt text are captured in the returned stdout. Input defaults to an
	// empty reader unless a test provided one via testStdin (runOchamiWithInput).
	stdin := io.Reader(strings.NewReader(""))
	if testStdin != nil {
		stdin = testStdin
		testStdin = nil
	}
	restoreIos := cli.SetIOStream(stdin, w, w)
	defer restoreIos()

	rootCmd := NewRootCmd()
	rootCmd.SetArgs(args)
	// Capture Cobra output as well as command output. This lets tests verify
	// metacommand usage while preserving a single output assertion surface.
	rootCmd.SetOut(w)
	rootCmd.SetErr(w)

	runErr := rootCmd.Execute()

	// Restore os.Stdout and collect captured output.
	_ = w.Close()
	os.Stdout = origStdout
	captured := <-outCh
	_ = r.Close()

	return cmdResult{
		err:      runErr,
		exitCode: cli.ExitCode(runErr),
		stdout:   captured,
	}
}

// runOchamiWithInput runs the CLI with a scripted interactive stdin. The prompt
// text the command writes is captured in the returned cmdResult's stdout (the
// harness routes cli.Ios output into the same capture buffer).
//
// DEPRECATED: Use runOchamiWithInputAndRuntime() instead for test isolation.
// This function is kept for backward compatibility during migration.
func runOchamiWithInput(t *testing.T, input string, args ...string) cmdResult {
	t.Helper()
	testStdin = strings.NewReader(input)
	return runOchami(t, args...)
}

// runOchamiWithRuntime executes the ochami root command with an isolated Runtime,
// enabling test isolation and parallel test execution. Each test gets its own Runtime
// instance with isolated I/O streams, eliminating the need for global state manipulation.
//
// This is the runtime-based replacement for runOchami(). Once all tests are migrated
// to use this function, the old runOchami() and its global state (stdoutMu, testStdin)
// can be removed.
//
// Callers should generally include "--ignore-config" so the command does not
// read or create real config files, and "--uri <server.URL>" (on commands that
// accept it) to target an httptest.Server.
func runOchamiWithRuntime(t *testing.T, args ...string) cmdResult {
	t.Helper()

	// Create isolated runtime for this test with custom I/O streams
	// Use a single buffer for both stdout and stderr to match the behavior of runOchami()
	// which captures both in the same buffer. This is important for tests that check
	// for interactive prompts (written to stderr via rt.Ios.LoopYesNo) in the stdout.
	var combinedBuf bytes.Buffer
	stdinReader := strings.NewReader("")

	// Create runtime with both stdout and stderr pointing to the same buffer
	rt := cli.NewTestRuntime(stdinReader, &combinedBuf, &combinedBuf)

	// Temporarily redirect global I/O streams to the runtime streams for backward compatibility
	// This ensures that commands that still use cli.Ios.Out() instead of rt.Ios.Out() will
	// have their output captured correctly during the transition period.
	restoreIos := cli.SetIOStream(rt.Ios.In(), rt.Ios.Out(), rt.Ios.Err())
	defer restoreIos()

	// Create root command with runtime in context
	rootCmd := NewRootCmd()
	rootCmd.SetContext(rt.WithContext(context.Background()))
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
//
// This is the runtime-based replacement for runOchamiWithInput(). Once all tests are
// migrated to use this function, the old runOchamiWithInput() can be removed.
func runOchamiWithInputAndRuntime(t *testing.T, input string, args ...string) cmdResult {
	t.Helper()

	// Create isolated runtime for this test with custom I/O streams
	// Use a single buffer for both stdout and stderr to match the behavior of runOchami()
	// which captures both in the same buffer. This is important for tests that check
	// for interactive prompts (written to stderr via rt.Ios.LoopYesNo) in the stdout.
	var combinedBuf bytes.Buffer
	stdinReader := strings.NewReader(input)

	// Create runtime with both stdout and stderr pointing to the same buffer
	rt := cli.NewTestRuntime(stdinReader, &combinedBuf, &combinedBuf)

	// Temporarily redirect global I/O streams to the runtime streams for backward compatibility
	// This ensures that commands that still use cli.Ios.Out() instead of rt.Ios.Out() will
	// have their output captured correctly during the transition period.
	restoreIos := cli.SetIOStream(rt.Ios.In(), rt.Ios.Out(), rt.Ios.Err())
	defer restoreIos()

	// Create root command with runtime in context
	rootCmd := NewRootCmd()
	rootCmd.SetContext(rt.WithContext(context.Background()))
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
	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()

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
	// TODO: Enable t.Parallel() once race conditions are resolved
	// t.Parallel()

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
