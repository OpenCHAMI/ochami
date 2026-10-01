// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"context"
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
