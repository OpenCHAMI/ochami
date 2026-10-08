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

// TestApplyFormatFlags_Success verifies that ApplyFormatFlags sets the
// runtime's input and output formats from --format-input and --format-output.
func TestApplyFormatFlags_Success(t *testing.T) {
	t.Parallel()

	// Create a command with format flags
	cmd := &cobra.Command{}
	cmd.Flags().String("format-input", "json", "")
	cmd.Flags().String("format-output", "yaml", "")

	// Set both flags, which also marks them as changed
	if err := cmd.Flags().Set("format-input", "yaml"); err != nil {
		t.Fatalf("set format-input: %v", err)
	}
	if err := cmd.Flags().Set("format-output", "json"); err != nil {
		t.Fatalf("set format-output: %v", err)
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

// TestApplyFormatFlags_Unchanged verifies that unchanged flags don't modify runtime.
func TestApplyFormatFlags_Unchanged(t *testing.T) {
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
