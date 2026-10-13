// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"testing"
	"time"

	"github.com/spf13/pflag"
)

// TestAddPositiveDurationFlag_AcceptsPositive verifies that the flag keeps its
// default until set, accepts a positive duration, and reads back with
// GetDuration.
func TestAddPositiveDurationFlag_AcceptsPositive(t *testing.T) {
	t.Parallel()

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	AddPositiveDurationFlag(fs, "timeout", 30*time.Second, "")
	if got, err := fs.GetDuration("timeout"); err != nil || got != 30*time.Second {
		t.Fatalf("GetDuration() before parsing = %v, %v; want 30s, nil", got, err)
	}
	if got := fs.Lookup("timeout").DefValue; got != "30s" {
		t.Errorf("DefValue = %q, want %q", got, "30s")
	}

	if err := fs.Parse([]string{"--timeout", "1m30s"}); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got, err := fs.GetDuration("timeout"); err != nil || got != 90*time.Second {
		t.Fatalf("GetDuration() = %v, %v; want 1m30s, nil", got, err)
	}
}
