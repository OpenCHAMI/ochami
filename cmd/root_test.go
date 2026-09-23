// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// root_test.go exercises handleExecuteError, the testable core of Execute that
// resolves an error to a process exit code and emits the help hint, without
// terminating the test binary via os.Exit, plus general root-command-tree
// dispatch behavior such as every metacommand printing usage when invoked
// with no further subcommand.

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/openchami/ochami/internal/cli"
)

func TestHandleExecuteError(t *testing.T) {
	rootCmd := NewRootCmd()

	// nil error -> success.
	if code := handleExecuteError(rootCmd, nil); code != cli.CodeSuccess {
		t.Errorf("handleExecuteError(nil) = %d, want %d", code, cli.CodeSuccess)
	}

	// A coded error resolves to its code and emits the help hint.
	err := cli.Errorf(cli.CodeHTTP, "boom")
	if code := handleExecuteError(rootCmd, err); code != cli.CodeHTTP {
		t.Errorf("handleExecuteError(CodeHTTP err) = %d, want %d", code, cli.CodeHTTP)
	}

	// A plain error resolves to the generic code.
	plain := cli.Errorf(cli.CodeGeneric, "plain")
	if code := handleExecuteError(rootCmd, plain); code != cli.CodeGeneric {
		t.Errorf("handleExecuteError(generic err) = %d, want %d", code, cli.CodeGeneric)
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

// TestMetacommandPathsPrintUsage verifies every metacommand in the tree prints
// its usage when invoked with no further subcommand or arguments.
func TestMetacommandPathsPrintUsage(t *testing.T) {
	t.Parallel()

	paths := [][]string{
		{},
		{"boot"}, {"boot", "bmc"}, {"boot", "config"}, {"boot", "node"}, {"boot", "service"},
		{"bss"}, {"bss", "boot"}, {"bss", "boot", "image"}, {"bss", "boot", "params"},
		{"bss", "boot", "script"}, {"bss", "hosts"}, {"bss", "service"},
		{"cloud-init"}, {"cloud-init", "defaults"}, {"cloud-init", "group"}, {"cloud-init", "group", "get"}, {"cloud-init", "node"}, {"cloud-init", "node", "get"}, {"cloud-init", "service"},
		{"config"}, {"config", "cluster"}, {"discover"},
		{"metadata"}, {"metadata", "defaults"}, {"metadata", "group"}, {"metadata", "instance"}, {"metadata", "peer"}, {"metadata", "service"},
		{"pcs"}, {"pcs", "service"}, {"pcs", "status"}, {"pcs", "transition"},
		{"rcs"}, {"rcs", "console"}, {"rcs", "service"},
		{"smd"}, {"smd", "compep"}, {"smd", "component"}, {"smd", "group"}, {"smd", "group", "member"},
		{"smd", "iface"}, {"smd", "rfe"}, {"smd", "service"},
	}

	for _, path := range paths {
		name := "root"
		if len(path) > 0 {
			name = strings.Join(path, " ")
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			args := append([]string{"--ignore-config"}, path...)
			res := runOchamiWithRuntime(t, args...)
			if res.err != nil {
				t.Fatalf("unexpected error: %v", res.err)
			}
			if !strings.Contains(res.stdout, "Usage:") {
				t.Errorf("stdout = %q, want usage", res.stdout)
			}
		})
	}
}
