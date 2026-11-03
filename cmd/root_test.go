// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// root_test.go exercises handleExecuteError, the testable core of Execute that
// resolves an error to a process exit code and emits the help hint, without
// terminating the test binary via os.Exit, and checks that apply to the root
// command tree as a whole. It also checks that the root command loads the
// config file and resolves a command's base URI and token from its cluster.

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/format"
)

// TestHandleExecuteError verifies that handleExecuteError maps a nil error to
// CodeSuccess and a coded error (HTTP, generic, declined, or interrupted) to
// its code.
func TestHandleExecuteError(t *testing.T) {
	t.Parallel()

	rootCmd := NewRootCmd()

	// nil error -> success.
	if code := handleExecuteError(rootCmd, nil); code != cli.CodeSuccess {
		t.Errorf("handleExecuteError(nil) = %d, want %d (%s)", code, cli.CodeSuccess, cli.CodeName(cli.CodeSuccess))
	}

	// A coded error resolves to its code and emits the help hint.
	err := cli.Errorf(cli.CodeHTTP, "boom")
	if code := handleExecuteError(rootCmd, err); code != cli.CodeHTTP {
		t.Errorf("handleExecuteError(%s err) = %d, want %d", cli.CodeName(cli.CodeHTTP), code, cli.CodeHTTP)
	}

	// A plain error resolves to the generic code.
	plain := cli.Errorf(cli.CodeGeneric, "plain")
	if code := handleExecuteError(rootCmd, plain); code != cli.CodeGeneric {
		t.Errorf("handleExecuteError(generic err) = %d, want %d (%s)", code, cli.CodeGeneric, cli.CodeName(cli.CodeGeneric))
	}

	// A declined prompt resolves to CodeDeclined.
	declined := cli.Errorf(cli.CodeDeclined, "user declined")
	if code := handleExecuteError(rootCmd, declined); code != cli.CodeDeclined {
		t.Errorf("handleExecuteError(declined err) = %d, want %d (%s)", code, cli.CodeDeclined, cli.CodeName(cli.CodeDeclined))
	}

	// An interruption resolves to CodeInterrupted.
	interrupted := cli.Interrupted(signalledContext(os.Interrupt), context.Canceled)
	if code := handleExecuteError(rootCmd, interrupted); code != cli.CodeInterrupted {
		t.Errorf("handleExecuteError(interrupted err) = %d, want %d (%s)", code, cli.CodeInterrupted, cli.CodeName(cli.CodeInterrupted))
	}
}

// signalledContext returns a context cancelled as cli.NotifyContext cancels
// one when sig arrives.
func signalledContext(sig os.Signal) context.Context {
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(&cli.InterruptError{Signal: sig})
	return ctx
}

// TestExecute_Interrupted verifies that a signal cancelling the context stops
// a command blocked in a request, a confirmation prompt, a stdin payload read,
// or a console websocket handshake promptly, and that it exits with the
// signal's code rather than the code of the error the cancelled operation
// returned: CodeInterrupted for SIGINT and CodeTerminated for SIGTERM.
func TestExecute_Interrupted(t *testing.T) {
	t.Parallel()

	var deletes atomic.Int32
	// The server never answers, so a request blocks until it is cancelled.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletes.Add(1)
		}
		<-r.Context().Done()
	}))
	// The subtests run in parallel, after this function returns.
	t.Cleanup(func() {
		srv.Close()
		if n := deletes.Load(); n != 0 {
			t.Errorf("DELETE count = %d, want 0: an interrupted confirmation must not delete", n)
		}
	})

	tests := []struct {
		name     string
		sig      os.Signal
		args     []string
		wantCode int
	}{
		{name: "request", sig: os.Interrupt, args: []string{"smd", "component", "get"}, wantCode: cli.CodeInterrupted},
		{name: "request SIGTERM", sig: syscall.SIGTERM, args: []string{"smd", "component", "get"}, wantCode: cli.CodeTerminated},
		{name: "delete prompt", sig: os.Interrupt, args: []string{"smd", "group", "delete", "compute"}, wantCode: cli.CodeInterrupted},
		{name: "stdin payload", sig: os.Interrupt, args: []string{"smd", "component", "add", "-d", "@-"}, wantCode: cli.CodeInterrupted},
		{name: "console show handshake", sig: os.Interrupt, args: []string{"rcs", "console", "show", "-f", "x0"}, wantCode: cli.CodeInterrupted},
		{name: "console connect handshake", sig: os.Interrupt, args: []string{"rcs", "console", "connect", "x0"}, wantCode: cli.CodeInterrupted},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// No one ever writes to stdin, so prompts and stdin reads block
			// until they are cancelled.
			stdin, stdinWriter := io.Pipe()
			defer stdinWriter.Close()
			var out bytes.Buffer
			rt := cli.NewTestRuntime(stdin, &out, &out)
			rootCmd := NewRootCmd()
			rootCmd.SetContext(cli.ContextWithRuntime(context.Background(), rt))
			rootCmd.SetArgs(append([]string{"--ignore-config"}, append(tc.args, "--uri", srv.URL, "--token", "t")...))
			rootCmd.SetOut(&out)
			rootCmd.SetErr(&out)

			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			time.AfterFunc(50*time.Millisecond, func() { cancel(&cli.InterruptError{Signal: tc.sig}) })

			done := make(chan int, 1)
			go func() { done <- execute(ctx, rootCmd) }()
			select {
			case code := <-done:
				if code != tc.wantCode {
					t.Errorf("execute() = %d (%s), want %d (%s)\noutput:\n%s", code, cli.CodeName(code), tc.wantCode, cli.CodeName(tc.wantCode), out.String())
				}
			case <-time.After(5 * time.Second):
				t.Fatal("execute() did not return within 5s of the signal")
			}
		})
	}
}

// TestNoDuplicateFlags ensures that no flag is redefined along the same
// command path in the tree. It checks for two types of duplicate flag definitions:
//
//  1. Local flags that shadow inherited persistent flags (e.g., a flag defined
//     as PersistentFlag in a parent command and as a local Flag in a child command)
//
//  2. Persistent flags that shadow inherited persistent flags (e.g., a flag defined
//     as PersistentFlag in both a parent and a child command)
//
// These duplicates can lead to unexpected behavior and should be caught at
// test time since Cobra does not catch them at compile time.
//
// Note: Duplicate flags in different command subtrees (e.g., both 'boot' and
// 'metadata' defining a 'timeout' persistent flag) are intentional and allowed,
// as each subtree maintains its own flag namespace.
func TestNoDuplicateFlags(t *testing.T) {
	t.Parallel()

	rootCmd := NewRootCmd()

	// Recursively traverse all commands in the tree, passing inherited persistent flags
	// from parent to child.
	var checkCommand func(cmd *cobra.Command, inheritedPersistent map[string]string)
	checkCommand = func(cmd *cobra.Command, inheritedPersistent map[string]string) {
		cmdPath := cmd.CommandPath()

		// Collect persistent flags defined at this specific command level only.
		// We determine this by checking which flags in PersistentFlags are NOT
		// in the inherited set (i.e., they were added at this level)
		localPersistentFlags := make(map[string]bool)
		cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
			// Skip internal flags added by Cobra
			if f.Name == "help" || f.Name == "version" {
				return
			}
			// Only include if not inherited from parent
			if _, exists := inheritedPersistent[f.Name]; !exists {
				localPersistentFlags[f.Name] = true
			}
		})

		// Collect local flags defined at this specific command level
		localFlags := make(map[string]bool)
		cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
			// Skip internal flags added by Cobra
			if f.Name == "help" || f.Name == "version" {
				return
			}
			localFlags[f.Name] = true
		})

		// Check for conflicts: local flags that shadow inherited persistent flags
		// This is the bug we want to catch (e.g., envelope flag redefined in child)
		for flagName := range localFlags {
			if existingCmd, exists := inheritedPersistent[flagName]; exists {
				t.Errorf("Flag '%s' redefined as local flag at %s (already defined as persistent at %s)",
					flagName, cmdPath, existingCmd)
			}
		}

		// Check for conflicts: persistent flags that shadow inherited persistent flags
		// along the same path (e.g., parent and child both defining the same flag)
		for flagName := range localPersistentFlags {
			if existingCmd, exists := inheritedPersistent[flagName]; exists {
				t.Errorf("Flag '%s' redefined as persistent flag at %s (already defined as persistent at %s)",
					flagName, cmdPath, existingCmd)
			}
		}

		// Update inherited persistent flags for child commands
		// Create a new map to avoid modifying the parent's map
		newInherited := make(map[string]string)
		for k, v := range inheritedPersistent {
			newInherited[k] = v
		}
		for flagName := range localPersistentFlags {
			newInherited[flagName] = cmdPath
		}

		// Recursively check child commands with updated inherited flags
		for _, subCmd := range cmd.Commands() {
			checkCommand(subCmd, newInherited)
		}
	}

	// Start with root command's persistent flags
	rootPersistent := make(map[string]string)
	rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if f.Name == "help" || f.Name == "version" {
			return
		}
		rootPersistent[f.Name] = "ochami"
	})

	// Check all top-level commands (children of root) with root's persistent flags
	for _, subCmd := range rootCmd.Commands() {
		checkCommand(subCmd, rootPersistent)
	}
}

// TestServiceURIFlagUsage verifies that the --uri flag of every service
// command names the service that command talks to in its help text.
func TestServiceURIFlagUsage(t *testing.T) {
	t.Parallel()

	services := map[string]string{
		"boot":       "boot service",
		"bss":        "BSS",
		"cloud-init": "cloud-init",
		"metadata":   "metadata service",
		"pcs":        "PCS",
		"rcs":        "remote-console",
		"smd":        "SMD",
	}

	rootCmd := NewRootCmd()
	for _, cmd := range rootCmd.Commands() {
		flag := cmd.PersistentFlags().Lookup("uri")
		if flag == nil {
			continue
		}
		service, ok := services[cmd.Name()]
		if !ok {
			t.Errorf("command %q has a --uri flag but no expected service name in this test", cmd.Name())
			continue
		}
		delete(services, cmd.Name())

		want := "absolute base URI or relative base path of " + service
		if flag.Usage != want {
			t.Errorf("%s --uri usage = %q, want %q", cmd.CommandPath(), flag.Usage, want)
		}
	}
	for name := range services {
		t.Errorf("command %q not found or has no --uri flag", name)
	}
}

// TestRootCommand_ConcurrentRuntimesStayIsolated verifies that command trees
// run concurrently, each with its own runtime, all succeed, write only to
// their own runtime's output stream, and leave each runtime's token
// unchanged.
func TestRootCommand_ConcurrentRuntimesStayIsolated(t *testing.T) {
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
		if res.exitCode != cli.CodeSuccess {
			t.Errorf("runtime %d: exit code = %d, want %d (%s)", i, res.exitCode, cli.CodeSuccess, cli.CodeName(cli.CodeSuccess))
		}
		if n := strings.Count(res.stdout, "Version:"); n != 1 {
			t.Errorf("runtime %d: output has %d version reports, want 1: %q", i, n, res.stdout)
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

// TestDefaultCluster_URIResolution verifies a command resolves its base URI from
// the default cluster's cluster.uri in a config file (no --uri flag).
func TestDefaultCluster_URIResolution(t *testing.T) {
	t.Parallel()

	var hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	cfg := writeTempConfig(t, `default-cluster: demo
clusters:
- name: demo
  cluster:
    uri: `+srv.URL+`
    enable-auth: false
`)

	res := runOchamiWithRuntime(t, "--config", cfg, "smd", "group", "get")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !hit {
		t.Error("expected the server (from default-cluster uri) to be contacted")
	}
}

// TestPerServiceURI_Override verifies a per-service URI override in the cluster
// config is honored.
func TestPerServiceURI_Override(t *testing.T) {
	t.Parallel()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	cfg := writeTempConfig(t, `default-cluster: demo
clusters:
- name: demo
  cluster:
    uri: https://unused.example.com
    smd:
      uri: `+srv.URL+`/smd
    enable-auth: false
`)

	res := runOchamiWithRuntime(t, "--config", cfg, "smd", "group", "get")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.HasPrefix(gotPath, "/smd") {
		t.Errorf("path = %q, want the /smd override", gotPath)
	}
}

// TestEnableAuth_ReadsTokenFromEnv verifies that, for a cluster with
// enable-auth set, a command reads the token from <CLUSTER>_ACCESS_TOKEN and
// sends it as a bearer token.
func TestEnableAuth_ReadsTokenFromEnv(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	cfg := writeTempConfig(t, `default-cluster: demo
clusters:
- name: demo
  cluster:
    uri: `+srv.URL+`
    enable-auth: true
`)

	tok := validToken(t)
	t.Setenv("DEMO_ACCESS_TOKEN", tok)

	res := runOchamiWithRuntimeEnv(t, cli.EnvironmentFunc(os.LookupEnv), "--config", cfg, "smd", "group", "get")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if !strings.Contains(gotAuth, "Bearer") {
		t.Errorf("Authorization header = %q, want it to carry the bearer token", gotAuth)
	}
}

// TestEnableAuth_DisabledSkipsToken verifies that with enable-auth false, no
// token is required or sent.
func TestEnableAuth_DisabledSkipsToken(t *testing.T) {
	t.Parallel()

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	cfg := writeTempConfig(t, `default-cluster: demo
clusters:
- name: demo
  cluster:
    uri: `+srv.URL+`
    enable-auth: false
`)

	res := runOchamiWithRuntime(t, "--config", cfg, "smd", "group", "get")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	if gotAuth != "" {
		t.Errorf("Authorization header = %q, want empty (auth disabled)", gotAuth)
	}
}
