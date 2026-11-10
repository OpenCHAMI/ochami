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
	"time"

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

// gatedReader blocks every Read until a value arrives on its channel, then
// returns that value, simulating a terminal no one is typing into.
type gatedReader struct{ lines chan string }

func (r gatedReader) Read(p []byte) (int, error) {
	line, ok := <-r.lines
	if !ok {
		return 0, io.EOF
	}
	return copy(p, line), nil
}

// cancelDeadline bounds how long a cancelled read may take to return.
const cancelDeadline = 5 * time.Second

// TestIOStreams_LoopYesNoCancelled verifies that cancelling the context
// returns a prompt blocked on input with context.Canceled, rather than an
// answer of no, and ends the prompt line.
func TestIOStreams_LoopYesNoCancelled(t *testing.T) {
	t.Parallel()

	errBuf := &bytes.Buffer{}
	ios := NewIOStreams(gatedReader{make(chan string)}, io.Discard, errBuf)
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(10*time.Millisecond, cancel)

	done := make(chan error, 1)
	go func() {
		_, err := ios.LoopYesNo(ctx, "Proceed?")
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("LoopYesNo() error = %v, want context.Canceled", err)
		}
	case <-time.After(cancelDeadline):
		t.Fatal("LoopYesNo() did not return after its context was cancelled")
	}
	if got, want := errBuf.String(), "Proceed? [yn]:\n"; got != want {
		t.Errorf("stderr = %q, want %q", got, want)
	}
}

// TestIOStreams_InContextCancelled verifies that a Read abandoned when the
// context is cancelled delivers its input to the next Read instead of losing
// it.
func TestIOStreams_InContextCancelled(t *testing.T) {
	t.Parallel()

	in := gatedReader{make(chan string, 1)}
	ios := NewIOStreams(in, io.Discard, io.Discard)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	r := ios.InContext(ctx)
	buf := make([]byte, 8)
	if _, err := r.Read(buf); !errors.Is(err, context.Canceled) {
		t.Fatalf("Read() error = %v, want context.Canceled", err)
	}

	// The abandoned Read is still waiting on input; a reader with a live
	// context sharing its state must not lose what it eventually reads.
	cr := r.(*contextReader)
	cr.ctx = context.Background()
	in.lines <- "late"
	n, err := r.Read(buf)
	if err != nil || string(buf[:n]) != "late" {
		t.Errorf("Read() = %q, %v; want %q, nil", buf[:n], err, "late")
	}
}

// TestIOStreams_Errors verifies that LoopYesNo returns prompt-write and
// input-read failures, and that WriteOutput reports a writer that writes too
// few or too many bytes as io.ErrShortWrite.
func TestIOStreams_Errors(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("stream failure")
	t.Run("prompt writer", func(t *testing.T) {
		ios := NewIOStreams(strings.NewReader("y\n"), io.Discard, errorWriter{sentinel})
		if _, err := ios.LoopYesNo(context.Background(), "Proceed?"); !errors.Is(err, sentinel) {
			t.Fatalf("LoopYesNo() error = %v, want stream failure", err)
		}
	})
	t.Run("input reader", func(t *testing.T) {
		ios := NewIOStreams(errorReader{sentinel}, io.Discard, io.Discard)
		if _, err := ios.LoopYesNo(context.Background(), "Proceed?"); !errors.Is(err, sentinel) {
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
