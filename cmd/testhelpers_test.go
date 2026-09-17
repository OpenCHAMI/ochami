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
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

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
		_, _ = io.Copy(&buf, r)
		outCh <- buf.String()
	}()

	// Reset the token global between runs so a token set by a previous test
	// does not leak into this one.
	cli.Token = ""

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
	// Route Cobra's own output (usage/errors) away from os.Stdout so it does
	// not pollute captured command output; command output uses fmt.Print
	// which goes to the real os.Stdout we swapped above.
	rootCmd.SetOut(io.Discard)
	rootCmd.SetErr(io.Discard)

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
