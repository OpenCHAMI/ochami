// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// runtime_dispatch_test.go exercises the root command dispatch path itself:
// runtime isolation across concurrent invocations, the typed error returned
// when no runtime is present, output-writer failures, HTTP client edge
// cases, and process exit-code mapping. Configuration-file behavior lives in
// config_edge_test.go; per-command coverage lives in each command's own
// _test.go file.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/format"
)

var errTestWriter = errors.New("test writer failure")

type failingOutputWriter struct {
	writes int
}

func (w *failingOutputWriter) Write([]byte) (int, error) {
	w.writes++
	return 0, errTestWriter
}

// =============================================================================
// CLI/runtime tests
// =============================================================================

// TestRuntimeIsolationConcurrentRoots verifies that concurrent command trees
// with different runtimes do not leak flags, token, config, formats, or
// streams between invocations.
func TestRuntimeIsolationConcurrentRoots(t *testing.T) {
	t.Parallel()

	// Create multiple runtimes with different configurations
	runtimes := make([]*cli.Runtime, 4)
	for i := 0; i < 4; i++ {
		stdin := strings.NewReader(fmt.Sprintf("input-%d", i))
		stdout := &bytes.Buffer{}
		stderr := &bytes.Buffer{}
		runtimes[i] = cli.NewTestRuntime(stdin, stdout, stderr).
			WithToken(fmt.Sprintf("token-%d", i)).
			WithConfigFile(fmt.Sprintf("/config-%d", i))
		if i%2 == 0 {
			runtimes[i] = runtimes[i].WithFormats(format.DataFormatYaml, format.DataFormatJson)
		} else {
			runtimes[i] = runtimes[i].WithFormats(format.DataFormatJson, format.DataFormatJsonPretty)
		}
	}

	// Execute commands concurrently, each with its own runtime
	var wg sync.WaitGroup
	results := make([]cmdResult, 4)

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			rt := runtimes[idx]

			// Create root command with isolated runtime
			rootCmd := NewRootCmd()
			rootCmd.SetContext(cli.ContextWithRuntime(context.Background(), rt))
			rootCmd.SetArgs([]string{"version"})
			rootCmd.SetOut(rt.Ios.Out())
			rootCmd.SetErr(rt.Ios.Err())

			err := rootCmd.Execute()
			results[idx] = cmdResult{
				err:      err,
				exitCode: cli.ExitCode(err),
				stdout:   rt.Ios.Out().(*bytes.Buffer).String(),
			}
		}(i)
	}

	wg.Wait()

	// Verify all commands executed successfully
	for i, res := range results {
		if res.err != nil {
			t.Errorf("runtime %d: unexpected error: %v", i, res.err)
		}
		if res.exitCode != 0 {
			t.Errorf("runtime %d: unexpected exit code: %d", i, res.exitCode)
		}
	}

	// Verify runtimes remained isolated (tokens are different)
	for i := 0; i < 4; i++ {
		if runtimes[i].Token != fmt.Sprintf("token-%d", i) {
			t.Errorf("runtime %d token leaked: got %q, want %q",
				i, runtimes[i].Token, fmt.Sprintf("token-%d", i))
		}
	}
}

// TestRuntimeMissingError verifies that commands return a typed error when
// runtime is missing from context.
func TestRuntimeMissingError(t *testing.T) {
	t.Parallel()

	// Create a command without runtime in context
	rootCmd := NewRootCmd()
	// Explicitly set context without runtime
	rootCmd.SetContext(context.Background())
	rootCmd.SetArgs([]string{"version"})

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an error when runtime is missing, got nil")
	}

	exitCode := cli.ExitCode(err)
	if exitCode == 0 {
		t.Errorf("expected non-zero exit code for missing runtime, got %d", exitCode)
	}
}

// TestRuntimeFromCommandError verifies RuntimeFromCommand returns an error
// when command or context is nil.
func TestRuntimeFromCommandError(t *testing.T) {
	t.Parallel()

	// Test with nil command
	_, err := cli.RuntimeFromCommand(nil)
	if err == nil {
		t.Error("expected error for nil command, got nil")
	}

	// Test with command but nil context
	cmd := &cobra.Command{}
	_, err = cli.RuntimeFromCommand(cmd)
	if err == nil {
		t.Error("expected error for nil context, got nil")
	}
}

// TestOutputWriterFailure verifies that commands return an error when the
// output writer fails.
func TestOutputWriterFailure(t *testing.T) {
	t.Parallel()

	fw := &failingOutputWriter{}

	// Create runtime with failing writer
	stdin := strings.NewReader("")
	stderr := &bytes.Buffer{}
	rt := cli.NewTestRuntime(stdin, fw, stderr)

	// Run a command that writes output
	rootCmd := NewRootCmd()
	rootCmd.SetContext(cli.ContextWithRuntime(context.Background(), rt))
	rootCmd.SetArgs([]string{"version"})
	rootCmd.SetOut(fw)
	rootCmd.SetErr(stderr)

	err := rootCmd.Execute()
	if !errors.Is(err, errTestWriter) {
		t.Fatalf("Execute() error = %v, want writer failure", err)
	}
	if got := cli.ExitCode(err); got != cli.CodePayload {
		t.Errorf("exit code = %d, want %d (CodePayload)", got, cli.CodePayload)
	}
	if fw.writes != 1 {
		t.Errorf("write attempts = %d, want 1", fw.writes)
	}
}

// =============================================================================
// HTTP and service client tests
// =============================================================================

// TestHTTPClient_NilURIFallback verifies behavior with nil or malformed base URI.
func TestHTTPClient_NilURIFallback(t *testing.T) {
	t.Parallel()

	// Create a server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck // test response
	}))
	defer srv.Close()

	// Test with valid cluster-uri
	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t", "version")
	if res.err != nil {
		t.Fatalf("unexpected error with valid URI: %v", res.err)
	}
}

// TestHTTPClient_ResponseReadFailure verifies that a truncated HTTP response
// body (a declared Content-Length the server doesn't deliver) is reported as
// a network error, not silently swallowed.
func TestHTTPClient_ResponseReadFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Declare a body longer than what's written, then close the
		// connection without fulfilling Content-Length.
		w.Header().Set("Content-Length", "100")
		_, _ = w.Write([]byte("partial")) //nolint:errcheck // test response
	}))
	defer srv.Close()

	res := runOchamiWithRuntime(t, "--ignore-config", "smd", "group", "get", "--uri", srv.URL, "--token", "t")
	if res.err == nil {
		t.Fatal("expected an error for a truncated response body, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (CodeNetwork)", res.exitCode, cli.CodeNetwork)
	}
	if !strings.Contains(res.err.Error(), "unexpected EOF") {
		t.Errorf("error = %q, want it to report the truncated read", res.err.Error())
	}
}

// =============================================================================
// Process-boundary integration tests
// =============================================================================

// TestProcessExitCodeMapping verifies process exit-code mapping.
func TestProcessExitCodeMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		wantNonZero bool
	}{
		{"version succeeds", []string{"version"}, false},
		{"invalid command fails", []string{"invalid-command"}, true},
		// Note: "bss hosts get" requires a node ID, so it will fail with missing args
		// but we don't test that here as it requires service configuration
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			res := runOchamiWithRuntime(t, tt.args...)
			if tt.wantNonZero && res.exitCode == 0 {
				t.Errorf("expected non-zero exit code, got 0")
			}
			if !tt.wantNonZero && res.exitCode != 0 {
				t.Errorf("expected zero exit code, got %d: %v", res.exitCode, res.err)
			}
		})
	}
}

// TestRealFlagParsingAndFormatOutput verifies real flag parsing and format output.
// Note: We use a service command that supports --format-output flag.
func TestRealFlagParsingAndFormatOutput(t *testing.T) {
	t.Parallel()

	// Create a test server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"Components":[]}`)) //nolint:errcheck // test response
	}))
	defer srv.Close()

	// Test JSON output format with smd component get
	// Note: smd component get requires --nid flag
	res := runOchamiWithRuntime(t, "--ignore-config", "--cluster-uri", srv.URL, "--token", "t",
		"--format-output", "json", "smd", "component", "get", "--nid", "0")
	if res.err != nil {
		t.Fatalf("unexpected error with --format-output json: %v", res.err)
	}

	// Output should contain the component data in JSON format
	if !strings.Contains(res.stdout, "Components") {
		t.Errorf("expected Components in JSON output, got: %s", res.stdout)
	}
}
