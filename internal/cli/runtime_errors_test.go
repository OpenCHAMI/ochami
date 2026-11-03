// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
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
// malformed config file as an error. Giving that error its exit code is
// cmd/root.go's PersistentPreRunE's responsibility, not InitConfig's, so the
// command-tree tests check the code rather than this one.
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

type zeroWriter struct{}

func (zeroWriter) Write([]byte) (int, error) { return 0, nil }

type oversizedWriter struct{}

func (oversizedWriter) Write(p []byte) (int, error) { return len(p) + 1, nil }

type errorWriter struct{ err error }

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }

type errorReader struct{ err error }

func (r errorReader) Read([]byte) (int, error) { return 0, r.err }

// TestIOStreams_Errors verifies that LoopYesNo returns prompt-write and
// input-read failures, and that WriteOutput reports a writer that writes too
// few or too many bytes as io.ErrShortWrite.
func TestIOStreams_Errors(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("stream failure")
	t.Run("prompt writer", func(t *testing.T) {
		ios := NewIOStreams(strings.NewReader("y\n"), io.Discard, errorWriter{sentinel})
		if _, err := ios.LoopYesNo("Proceed?"); !errors.Is(err, sentinel) {
			t.Fatalf("LoopYesNo() error = %v, want stream failure", err)
		}
	})
	t.Run("input reader", func(t *testing.T) {
		ios := NewIOStreams(errorReader{sentinel}, io.Discard, io.Discard)
		if _, err := ios.LoopYesNo("Proceed?"); !errors.Is(err, sentinel) {
			t.Fatalf("LoopYesNo() error = %v, want stream failure", err)
		}
	})
	t.Run("zero write", func(t *testing.T) {
		if err := WriteOutput(zeroWriter{}, []byte("data")); !errors.Is(err, io.ErrShortWrite) {
			t.Fatalf("WriteOutput() error = %v, want io.ErrShortWrite", err)
		}
	})
	t.Run("oversized write", func(t *testing.T) {
		if err := WriteOutput(oversizedWriter{}, []byte("data")); !errors.Is(err, io.ErrShortWrite) {
			t.Fatalf("WriteOutput() error = %v, want io.ErrShortWrite", err)
		}
	})
}
