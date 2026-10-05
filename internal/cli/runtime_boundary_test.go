// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/config"
)

// TestRuntime_NilEnvironmentUsesProcess verifies that a runtime given a nil
// environment looks variables up in the process environment.
func TestRuntime_NilEnvironmentUsesProcess(t *testing.T) {
	rt := NewRuntime().WithLogger(zerolog.Nop()).WithEnvironment(nil)
	const key = "OCHAMI_RUNTIME_BOUNDARY_TEST"
	t.Setenv(key, "present")
	if value, ok := rt.lookupEnv(key); !ok || value != "present" {
		t.Fatalf("lookupEnv(%q) = (%q, %v)", key, value, ok)
	}
}

// TestRuntimeFromCommand_IsPureGetter verifies that RuntimeFromCommand does not
// validate or apply format flags itself; format-flag validation is
// PersistentPreRunE's responsibility (via ApplyFormatFlags, called once at
// the root), so RuntimeFromCommand succeeds even when the command's format
// flags hold a value ApplyFormatFlags would reject.
func TestRuntimeFromCommand_IsPureGetter(t *testing.T) {
	rt := NewTestRuntime(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("format-input", "json", "")
	cmd.Flags().String("format-output", "json", "")
	if err := cmd.Flags().Set("format-input", "toml"); err != nil {
		t.Fatal(err)
	}
	cmd.SetContext(ContextWithRuntime(context.Background(), rt))
	got, err := RuntimeFromCommand(cmd)
	if err != nil {
		t.Fatalf("RuntimeFromCommand() returned an error for an unvalidated format flag: %v", err)
	}
	if got != rt {
		t.Error("RuntimeFromCommand() did not return the runtime stored in the command's context")
	}
}

// TestGetTimeout_FallsBackWhenFlagTypeIsWrong verifies that GetTimeout falls
// back to the configured timeout when the command's --timeout flag isn't a
// duration flag.
func TestGetTimeout_FallsBackWhenFlagTypeIsWrong(t *testing.T) {
	rt := NewRuntime().WithConfig(config.Config{Timeout: 37 * time.Second})
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("timeout", "", "")
	if err := cmd.Flags().Set("timeout", "not-a-duration"); err != nil {
		t.Fatal(err)
	}
	if got := rt.GetTimeout(cmd); got != 37*time.Second {
		t.Fatalf("GetTimeout() = %s, want 37s", got)
	}
}
