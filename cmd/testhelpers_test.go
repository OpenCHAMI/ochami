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
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/format"
)

// cmdResult captures everything a command-level test needs to assert on after
// running the CLI: the error returned from Execute, the exit code that error
// resolves to, and whatever the command wrote to os.Stdout. Interactive
// prompt/error text written via cli.Ios is captured into the same stdout
// field (see runOchamiWithStdin's cli.SetIOStream call) rather than a
// separate stream, so an assertion against stdout may also match text a real
// terminal would show on stderr.
type cmdResult struct {
	err      error
	exitCode int
	stdout   string
}

// stdoutMu serializes tests that capture os.Stdout. Because commands print
// directly to os.Stdout (via fmt.Print), and Go runs tests within a package
// sequentially by default but subtests/parallel tests could interleave, we
// guard the global swap with a mutex. This also makes runOchamiWithStdin safe
// to call from a t.Parallel() test today (calls simply serialize through the
// lock, including the cli.Token reset below) — though none of the tests in
// this package currently do, since parallelizing them for real requires the
// per-invocation cli.Runtime from stack/coverage/14-runtime-core-and-command-migration.
var stdoutMu sync.Mutex

// runOchami executes the ochami root command with the provided arguments and
// an empty interactive input stream, capturing anything written to
// os.Stdout during execution. It returns a cmdResult with the command error,
// the exit code cli.ExitCode maps that error to, and the captured stdout.
//
// Callers should generally include "--ignore-config" so the command does not
// read or create real config files, and "--uri <server.URL>" (on commands that
// accept it) to target an httptest.Server.
func runOchami(t *testing.T, args ...string) cmdResult {
	t.Helper()
	return runOchamiWithStdin(t, strings.NewReader(""), args...)
}

// runOchamiWithStdin is runOchami with an explicit interactive input stream,
// for commands that prompt (e.g. delete confirmations). Passing stdin as a
// parameter, rather than through shared package state, keeps concurrent
// invocations from being able to observe or clobber each other's input.
func runOchamiWithStdin(t *testing.T, stdin io.Reader, args ...string) cmdResult {
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

	// Reset globals bound to flags via pflag.Value (cli.FormatInput/FormatOutput,
	// via Flags().VarP) between runs. Unlike cli.Token/CACertPath/Insecure/
	// ConfigFile (bound with StringVarP/BoolVar, which reset the underlying
	// variable to its default on every NewRootCmd() call as part of flag
	// registration), VarP only wraps the existing Value without resetting it,
	// so a prior test's "--format-output yaml" would otherwise persist and
	// silently affect every later test in this package that omits the flag.
	cli.Token = ""
	cli.FormatInput = format.DataFormatJson
	cli.FormatOutput = format.DataFormatJson

	// Known limitation: some commands bind other pflag.Value-typed flags the
	// same VarP way to a var scoped to their own subpackage rather than to
	// internal/cli (e.g. cmd/pcs/status's powerFilter/mgmtFilter), which is
	// unexported and so cannot be reset from here. No test in this package
	// currently exercises one of those flags with a non-default value, so
	// this is dormant, not observed; a general fix needs the per-invocation
	// cli.Runtime from stack/coverage/14-runtime-core-and-command-migration,
	// same as the os.Stdout swap below.

	// Redirect the interactive I/O stream to the same capture pipe so output
	// written via cli.Ios.Out() (e.g. "rcs console show") and any interactive
	// prompt text are captured in the returned stdout.
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

// TestRunOchami_ResetsFormatFlagsBetweenCalls verifies that a --format-output
// passed to one runOchami call does not leak into a later call that omits it.
// cli.FormatOutput/cli.FormatInput are bound via Flags().VarP, which — unlike
// cli.Token/CACertPath/Insecure/ConfigFile's StringVarP/BoolVar — only wraps
// the existing pflag.Value without resetting it on registration, so the
// underlying package variable would otherwise keep whatever a previous call
// last set it to.
//
// root.go's PersistentPreRunE independently reapplies the configured default
// format when the invoked command's own --format-output flag wasn't changed,
// but only for commands that register that flag at all (it looks the flag up
// on the command and no-ops if not found). "bss service version" reads
// cli.FormatOutput to format its response but does not expose the flag
// itself, so it depends entirely on the harness resetting the global; it's
// used here as the command that would observe a leak if the reset were
// removed (as "smd component get" also has the flag, that reset would mask
// the bug this test is meant to catch).
func TestRunOchami_ResetsFormatFlagsBetweenCalls(t *testing.T) {
	setupSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer setupSrv.Close()

	setupRes := runOchami(t, "smd", "component", "get", "--ignore-config", "--uri", setupSrv.URL, "--format-output", "yaml")
	if setupRes.err != nil {
		t.Fatalf("setup: unexpected error: %v (exit %d)", setupRes.err, setupRes.exitCode)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version":"1.0.0"}`))
	}))
	defer srv.Close()

	res := runOchami(t, "bss", "service", "version", "--ignore-config", "--uri", srv.URL)
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(res.stdout, `"version":"1.0.0"`) {
		t.Errorf("stdout = %q, want plain JSON (a leaked --format-output yaml from the setup call would render this as YAML instead)", res.stdout)
	}
}
