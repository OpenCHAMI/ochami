// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestRuntimeFromCommand_MissingRuntime verifies that RuntimeFromCommand
// reports CodeConfig, rather than panicking, for a nil command, a command
// with no context, and a command whose context carries no runtime.
func TestRuntimeFromCommand_MissingRuntime(t *testing.T) {
	t.Parallel()

	withoutRuntime := &cobra.Command{}
	withoutRuntime.SetContext(context.Background())

	tests := []struct {
		name string
		cmd  *cobra.Command
	}{
		{"nil command", nil},
		{"no context", &cobra.Command{}},
		{"context without runtime", withoutRuntime},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rt, err := RuntimeFromCommand(tt.cmd)
			if err == nil || rt != nil {
				t.Fatalf("RuntimeFromCommand() = (%v, %v), want (nil, error)", rt, err)
			}
			if code := ExitCode(err); code != CodeConfig {
				t.Errorf("exit code = %d, want %d (%s)", code, CodeConfig, CodeName(CodeConfig))
			}
		})
	}
}

// TestInitLogging_RejectsInvalidConfiguration verifies that InitLogging rejects
// an invalid configured log level.
func TestInitLogging_RejectsInvalidConfiguration(t *testing.T) {
	rt := NewTestRuntime(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	rt.Config.Log.Level = "not-a-level"
	rt.Config.Log.Format = "json"
	rt.Config.Log.Color = "off"
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("log-format", "", "")
	cmd.Flags().String("log-level", "", "")
	cmd.Flags().String("log-color", "", "")
	if err := rt.InitLogging(cmd); err == nil {
		t.Fatal("InitLogging() accepted invalid level")
	}
}

// TestInitConfig_RejectsMalformedFile verifies that InitConfig surfaces a
// malformed config file as an error. Classifying that error as CodeConfig is
// cmd/root.go's PersistentPreRunE's responsibility, not InitConfig's, so it's
// covered at the command-tree level rather than here.
func TestInitConfig_RejectsMalformedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(path, []byte("clusters: [\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rt := NewTestRuntime(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}).WithConfigFile(path)
	cmd := &cobra.Command{Use: "test"}
	if err := rt.InitConfig(cmd, false); err == nil {
		t.Fatal("InitConfig() accepted a malformed config file")
	}
}
