// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// testhelpers_test.go provides shared helpers for command-level (end-to-end)
// tests. These tests build the real root command tree, point commands at an
// httptest.Server via each service's --uri flag, and assert both the outbound
// request shape and the resolved process exit code (via cli.ExitCode) that the
// command's returned error maps to. Config file reading is disabled with
// --ignore-config so tests never touch a user's real configuration.

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func writeJSONResponse(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Errorf("encode test response: %v", err)
	}
}

// cmdResult captures everything a command-level test needs to assert on after
// running the CLI: the error returned from Execute, the exit code that error
// resolves to, and everything the command wrote to its output and error
// streams. Both streams (cli.Ios and Cobra's own output) are captured into
// the same stdout field rather than separate ones, so an assertion against
// stdout may also match text a real terminal would show on stderr.
type cmdResult struct {
	err      error
	exitCode int
	stdout   string
}

// globalsMu serializes command runs. Each run swaps cli.Ios and resets
// internal/cli's other package globals (cli.Token and the format flags), so
// concurrent runs would see each other's streams and flag values. The lock
// also makes runOchamiWithStdin safe to call from a t.Parallel() test (calls
// simply run one at a time), though none of the tests in this package do,
// since running them in parallel for real needs per-invocation state in
// place of internal/cli's package globals.
var globalsMu sync.Mutex

// runOchami executes the ochami root command with the provided arguments and
// an empty interactive input stream, capturing everything the command writes
// to its output and error streams. It returns a cmdResult with the command
// error, the exit code cli.ExitCode maps that error to, and the captured
// output.
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

	globalsMu.Lock()
	defer globalsMu.Unlock()

	// Capture command output through a pipe.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

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

	// Point the I/O stream at the capture pipe so output written via
	// cli.Ios.Out() and any interactive prompt text are captured in the
	// returned stdout.
	restoreIos := cli.SetIOStream(stdin, w, w)
	defer restoreIos()

	rootCmd := NewRootCmd()
	rootCmd.SetArgs(args)
	// Capture Cobra output as well as command output. This lets tests verify
	// metacommand usage while preserving a single output assertion surface.
	rootCmd.SetOut(w)
	rootCmd.SetErr(w)

	runErr := rootCmd.Execute()

	// Collect captured output.
	_ = w.Close()
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
		w.Write([]byte(`[]`))
	}))
	defer setupSrv.Close()

	setupRes := runOchami(t, "smd", "component", "get", "--ignore-config", "--uri", setupSrv.URL, "--format-output", "yaml")
	if setupRes.err != nil {
		t.Fatalf("setup: unexpected error: %v (exit %d)", setupRes.err, setupRes.exitCode)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"version":"1.0.0"}`))
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

// assertFormattedOutput checks that out shows the key/value pair in the given
// output format: compact JSON, indented JSON, or YAML.
func assertFormattedOutput(t *testing.T, format, out, key, value string) {
	t.Helper()
	var want string
	switch format {
	case "json":
		want = fmt.Sprintf("%q:%q", key, value)
	case "json-pretty":
		want = fmt.Sprintf("%q: %q", key, value)
	case "yaml":
		want = key + ": " + value
	default:
		t.Fatalf("unknown output format %q", format)
	}
	if !strings.Contains(out, want) {
		t.Errorf("format %s: output = %q, want it to contain %q", format, out, want)
	}
}
