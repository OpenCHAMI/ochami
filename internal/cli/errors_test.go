// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

// errors_test.go unit-tests the exit-code contract in errors.go: the CodedError
// type, the Errorf/CodeError constructors, the ExitCode resolver's precedence
// rules, and the WrapUsageErrors command-tree wiring.

import (
	"errors"
	"fmt"
	"testing"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/config"
	"github.com/openchami/ochami/pkg/client"
)

// TestErrorfCodeAndMessage verifies Errorf builds a CodedError that carries the
// requested code, preserves the message, and supports %w unwrapping.
func TestErrorfCodeAndMessage(t *testing.T) {
	sentinel := errors.New("root cause")
	err := Errorf(CodePayload, "wrapping: %w", sentinel)

	var ce *CodedError
	if !errors.As(err, &ce) {
		t.Fatalf("Errorf did not produce a *CodedError: %T", err)
	}
	if ce.Code() != CodePayload {
		t.Errorf("Code() = %d, want %d", ce.Code(), CodePayload)
	}
	if ce.Error() != "wrapping: root cause" {
		t.Errorf("Error() = %q, want %q", ce.Error(), "wrapping: root cause")
	}
	if !errors.Is(err, sentinel) {
		t.Error("errors.Is could not find the wrapped sentinel")
	}
}

// TestCodedErrorNilInnerMessage verifies a CodedError with no wrapped error
// still produces a message.
func TestCodedErrorNilInnerMessage(t *testing.T) {
	ce := &CodedError{code: CodeGeneric}
	if ce.Error() == "" {
		t.Error("Error() returned empty string for a nil inner error")
	}
	if ce.Unwrap() != nil {
		t.Error("Unwrap() should be nil when there is no inner error")
	}
}

// TestCodeErrorNil verifies CodeError returns nil for a nil error.
func TestCodeErrorNil(t *testing.T) {
	if CodeError(CodeHTTP, nil) != nil {
		t.Error("CodeError(_, nil) should return nil")
	}
}

// TestCodeErrorNoDoubleWrap verifies CodeError preserves an already-coded
// error's original code rather than overwriting it.
func TestCodeErrorNoDoubleWrap(t *testing.T) {
	inner := Errorf(CodePayload, "payload problem")
	wrapped := CodeError(CodeUsage, inner)

	if ExitCode(wrapped) != CodePayload {
		t.Errorf("ExitCode = %d, want %d (original code should be preserved)", ExitCode(wrapped), CodePayload)
	}
}

// TestExitCodeResolution exercises ExitCode's precedence rules across all
// classifications.
func TestExitCodeResolution(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"nil is success", nil, CodeSuccess},
		{"explicit coded error", Errorf(CodeAuth, "no token"), CodeAuth},
		{"wrapped coded error", fmt.Errorf("outer: %w", Errorf(CodeNetwork, "dial")), CodeNetwork},
		{"http sentinel", fmt.Errorf("req failed: %w", client.UnsuccessfulHTTPError), CodeHTTP},
		{"config sentinel (unknown cluster)", fmt.Errorf("bad: %w", config.ErrUnknownCluster{ClusterName: "x"}), CodeConfig},
		{"config sentinel (missing uri)", config.ErrMissingURI{Service: "smd"}, CodeConfig},
		{"plain error falls back to generic", errors.New("boom"), CodeGeneric},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitCode(tc.err); got != tc.want {
				t.Errorf("ExitCode = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestExitCodeCodedBeatsSentinel verifies that an explicit code wins even when
// the wrapped chain also contains a known sentinel.
func TestExitCodeCodedBeatsSentinel(t *testing.T) {
	// A coded error whose inner chain also wraps the HTTP sentinel: the
	// explicit code should take precedence over the sentinel mapping.
	err := Errorf(CodeNetwork, "transport: %w", client.UnsuccessfulHTTPError)
	if got := ExitCode(err); got != CodeNetwork {
		t.Errorf("ExitCode = %d, want %d (explicit code should win over sentinel)", got, CodeNetwork)
	}
}

// TestWrapUsageErrorsFlagError verifies that after WrapUsageErrors, a flag
// parsing error resolves to CodeUsage.
func TestWrapUsageErrorsFlagError(t *testing.T) {
	root := &cobra.Command{
		Use:           "root",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE:          func(c *cobra.Command, a []string) error { return nil },
	}
	WrapUsageErrors(root)

	root.SetArgs([]string{"--nope"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected a flag error, got nil")
	}
	if ExitCode(err) != CodeUsage {
		t.Errorf("ExitCode = %d, want %d (CodeUsage)", ExitCode(err), CodeUsage)
	}
}

// TestWrapUsageErrorsArgError verifies that after WrapUsageErrors, an
// argument-count violation from a built-in Args validator resolves to
// CodeUsage.
func TestWrapUsageErrorsArgError(t *testing.T) {
	root := &cobra.Command{
		Use:           "root",
		Args:          cobra.ExactArgs(2),
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE:          func(c *cobra.Command, a []string) error { return nil },
	}
	WrapUsageErrors(root)

	root.SetArgs([]string{"only-one"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected an argument error, got nil")
	}
	if ExitCode(err) != CodeUsage {
		t.Errorf("ExitCode = %d, want %d (CodeUsage)", ExitCode(err), CodeUsage)
	}
}

// TestWrapUsageErrorsPreservesExplicitCode verifies that an Args validator
// returning an explicit CodedError keeps its own code instead of being coerced
// to CodeUsage.
func TestWrapUsageErrorsPreservesExplicitCode(t *testing.T) {
	root := &cobra.Command{
		Use: "root",
		Args: func(c *cobra.Command, a []string) error {
			return Errorf(CodePayload, "custom validation failure")
		},
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE:          func(c *cobra.Command, a []string) error { return nil },
	}
	WrapUsageErrors(root)

	root.SetArgs(nil)
	err := root.Execute()
	if err == nil {
		t.Fatal("expected a validation error, got nil")
	}
	if ExitCode(err) != CodePayload {
		t.Errorf("ExitCode = %d, want %d (explicit code should be preserved)", ExitCode(err), CodePayload)
	}
}

// TestWrapUsageErrorsRecurses verifies WrapUsageErrors applies to subcommands,
// so an arg error on a child command also resolves to CodeUsage.
func TestWrapUsageErrorsRecurses(t *testing.T) {
	root := &cobra.Command{Use: "root", SilenceErrors: true, SilenceUsage: true}
	child := &cobra.Command{
		Use:  "child",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, a []string) error { return nil },
	}
	root.AddCommand(child)
	WrapUsageErrors(root)

	root.SetArgs([]string{"child", "unexpected"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected an argument error on the child, got nil")
	}
	if ExitCode(err) != CodeUsage {
		t.Errorf("ExitCode = %d, want %d (CodeUsage)", ExitCode(err), CodeUsage)
	}
}
