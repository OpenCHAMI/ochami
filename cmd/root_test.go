// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// root_test.go exercises handleExecuteError, the testable core of Execute that
// resolves an error to a process exit code and emits the help hint, without
// terminating the test binary via os.Exit, and checks that apply to the root
// command tree as a whole.

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/openchami/ochami/internal/cli"
	"github.com/openchami/ochami/pkg/format"
)

// TestHandleExecuteError verifies that handleExecuteError maps a nil error to
// CodeSuccess and a coded error (HTTP, generic, or declined) to its code.
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
