// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

// cli_format_test.go contains tests for format-related CLI functions.

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/format"
)

// TestApplyFormatFlagsSuccess verifies that ApplyFormatFlags works correctly
// with valid format flags.
func TestApplyFormatFlagsSuccess(t *testing.T) {
	t.Parallel()

	// Create a command with format flags
	cmd := &cobra.Command{}
	cmd.Flags().String("format-input", "json", "")
	cmd.Flags().String("format-output", "yaml", "")

	// Set the flags to changed
	if err := cmd.Flags().Set("format-input", "yaml"); err != nil {
		t.Fatalf("set format-input: %v", err)
	}
	if err := cmd.Flags().Set("format-output", "json"); err != nil {
		t.Fatalf("set format-output: %v", err)
	}

	// Mark flags as changed
	if f := cmd.Flag("format-input"); f != nil {
		f.Changed = true
	}
	if f := cmd.Flag("format-output"); f != nil {
		f.Changed = true
	}

	// Create a runtime
	rt := NewTestRuntime(strings.NewReader(""), &strings.Builder{}, &strings.Builder{})

	// Apply format flags
	err := ApplyFormatFlags(cmd, rt)
	if err != nil {
		t.Fatalf("ApplyFormatFlags failed: %v", err)
	}

	// Verify formats were set
	if rt.FormatInput != format.DataFormatYaml {
		t.Errorf("expected FormatInput to be YAML, got %v", rt.FormatInput)
	}
	if rt.FormatOutput != format.DataFormatJson {
		t.Errorf("expected FormatOutput to be JSON, got %v", rt.FormatOutput)
	}
}

// TestApplyFormatFlagsNilCommand verifies error handling for nil command.
func TestApplyFormatFlagsNilCommand(t *testing.T) {
	t.Parallel()

	rt := NewTestRuntime(strings.NewReader(""), &strings.Builder{}, &strings.Builder{})

	err := ApplyFormatFlags(nil, rt)
	if err == nil {
		t.Error("expected error for nil command, got nil")
	}

	if !strings.Contains(err.Error(), "cannot apply format flags without a command runtime") {
		t.Errorf("expected specific error message, got: %v", err)
	}
}

// TestApplyFormatFlagsNilRuntime verifies error handling for nil runtime.
func TestApplyFormatFlagsNilRuntime(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}

	err := ApplyFormatFlags(cmd, nil)
	if err == nil {
		t.Error("expected error for nil runtime, got nil")
	}

	if !strings.Contains(err.Error(), "cannot apply format flags without a command runtime") {
		t.Errorf("expected specific error message, got: %v", err)
	}
}

// TestApplyFormatFlagsInvalidFormat verifies error handling for invalid format values.
func TestApplyFormatFlagsInvalidFormat(t *testing.T) {
	t.Parallel()

	// Create a command with invalid format flag
	cmd := &cobra.Command{}
	cmd.Flags().String("format-input", "json", "")
	if err := cmd.Flags().Set("format-input", "invalid-format"); err != nil {
		t.Fatalf("set format-input: %v", err)
	}

	// Mark flag as changed
	if f := cmd.Flag("format-input"); f != nil {
		f.Changed = true
	}

	rt := NewTestRuntime(strings.NewReader(""), &strings.Builder{}, &strings.Builder{})

	err := ApplyFormatFlags(cmd, rt)
	if err == nil {
		t.Error("expected error for invalid format, got nil")
	}

	if !strings.Contains(err.Error(), "invalid input format") {
		t.Errorf("expected invalid format error, got: %v", err)
	}
}

func TestApplyFormatFlagsInvalidOutputFormat(t *testing.T) {
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

// TestApplyFormatFlagsUnchanged verifies that unchanged flags don't modify runtime.
func TestApplyFormatFlagsUnchanged(t *testing.T) {
	t.Parallel()

	// Create a command with format flags but don't change them
	cmd := &cobra.Command{}
	cmd.Flags().String("format-input", "json", "")
	cmd.Flags().String("format-output", "json", "")

	// Don't mark flags as changed
	rt := NewTestRuntime(strings.NewReader(""), &strings.Builder{}, &strings.Builder{})

	// Set initial formats
	rt.FormatInput = format.DataFormatYaml
	rt.FormatOutput = format.DataFormatYaml

	// Apply format flags
	err := ApplyFormatFlags(cmd, rt)
	if err != nil {
		t.Fatalf("ApplyFormatFlags failed: %v", err)
	}

	// Verify formats were NOT changed (since flags weren't changed)
	if rt.FormatInput != format.DataFormatYaml {
		t.Errorf("expected FormatInput to remain YAML, got %v", rt.FormatInput)
	}
	if rt.FormatOutput != format.DataFormatYaml {
		t.Errorf("expected FormatOutput to remain YAML, got %v", rt.FormatOutput)
	}
}
