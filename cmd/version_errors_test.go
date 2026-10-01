// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// version_errors_test.go covers the error arms of the version command.

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

var errTestWriter = errors.New("test writer failure")

type failingOutputWriter struct {
	writes int
}

func (w *failingOutputWriter) Write([]byte) (int, error) {
	w.writes++
	return 0, errTestWriter
}

// TestVersion_OutputWriteFailure verifies that version reports a failure to
// write its output as CodePayload after a single write attempt.
func TestVersion_OutputWriteFailure(t *testing.T) {
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
		t.Errorf("exit code = %d, want %d (%s)", got, cli.CodePayload, cli.CodeName(cli.CodePayload))
	}
	if fw.writes != 1 {
		t.Errorf("write attempts = %d, want 1", fw.writes)
	}
}
