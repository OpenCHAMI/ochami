// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"io"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// TestAddPositiveDurationFlag_RejectsInvalid verifies that a duration of zero
// or less, or one that doesn't parse, is a flag error that resolves to
// CodeUsage once WrapUsageErrors is applied.
func TestAddPositiveDurationFlag_RejectsInvalid(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"0", "0s", "-5s", "soon"} {
		t.Run(value, func(t *testing.T) {
			t.Parallel()

			cmd := &cobra.Command{Use: "test", RunE: func(*cobra.Command, []string) error { return nil }}
			AddPositiveDurationFlag(cmd.Flags(), "timeout", 30*time.Second, "")
			WrapUsageErrors(cmd)
			cmd.SetArgs([]string{"--timeout", value})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			err := cmd.Execute()
			if err == nil {
				t.Fatalf("Execute() with --timeout %s = nil, want error", value)
			}
			if got := ExitCode(err); got != CodeUsage {
				t.Errorf("ExitCode() = %d, want %d (%s)", got, CodeUsage, CodeName(CodeUsage))
			}
		})
	}
}
