// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestApplyFormatFlags_NilCommand verifies that ApplyFormatFlags rejects a nil
// command.
func TestApplyFormatFlags_NilCommand(t *testing.T) {
	t.Parallel()

	rt := NewTestRuntime(strings.NewReader(""), &strings.Builder{}, &strings.Builder{})

	err := ApplyFormatFlags(nil, rt)
	if err == nil {
		t.Fatal("expected error for nil command, got nil")
	}

	if !strings.Contains(err.Error(), "cannot apply format flags without a command runtime") {
		t.Errorf("expected specific error message, got: %v", err)
	}
}

// TestApplyFormatFlags_NilRuntime verifies that ApplyFormatFlags rejects a nil
// runtime.
func TestApplyFormatFlags_NilRuntime(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}

	err := ApplyFormatFlags(cmd, nil)
	if err == nil {
		t.Fatal("expected error for nil runtime, got nil")
	}

	if !strings.Contains(err.Error(), "cannot apply format flags without a command runtime") {
		t.Errorf("expected specific error message, got: %v", err)
	}
}

// TestApplyFormatFlags_InvalidInputFormat verifies that ApplyFormatFlags
// rejects an unknown --format-input value.
func TestApplyFormatFlags_InvalidInputFormat(t *testing.T) {
	t.Parallel()

	// Create a command with invalid format flag
	cmd := &cobra.Command{}
	cmd.Flags().String("format-input", "json", "")
	if err := cmd.Flags().Set("format-input", "invalid-format"); err != nil {
		t.Fatalf("set format-input: %v", err)
	}

	rt := NewTestRuntime(strings.NewReader(""), &strings.Builder{}, &strings.Builder{})

	err := ApplyFormatFlags(cmd, rt)
	if err == nil {
		t.Fatal("expected error for invalid format, got nil")
	}

	if !strings.Contains(err.Error(), "invalid input format") {
		t.Errorf("expected invalid format error, got: %v", err)
	}
}

// TestApplyFormatFlags_InvalidOutputFormat verifies that ApplyFormatFlags
// rejects an unknown --format-output value.
func TestApplyFormatFlags_InvalidOutputFormat(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	cmd.Flags().String("format-output", "json", "")
	if err := cmd.Flags().Set("format-output", "invalid-format"); err != nil {
		t.Fatalf("set format-output: %v", err)
	}

	rt := NewTestRuntime(strings.NewReader(""), &strings.Builder{}, &strings.Builder{})
	err := ApplyFormatFlags(cmd, rt)
	if err == nil {
		t.Fatal("expected error for invalid output format, got nil")
	}
	if !strings.Contains(err.Error(), "invalid output format") {
		t.Errorf("error = %q, want invalid output format", err)
	}
}
