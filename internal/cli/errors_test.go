// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

// errors_test.go unit-tests the exit-code contract in errors.go: the CodedError
// type, the Errorf/EnsureCode constructors, the ExitCode resolver's precedence
// rules, and the WrapUsageErrors command-tree wiring.

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/config"
)

// TestErrorf_CodeAndMessage verifies Errorf builds a CodedError that carries the
// requested code, preserves the message, and supports %w unwrapping.
func TestErrorf_CodeAndMessage(t *testing.T) {
	sentinel := errors.New("root cause")
	err := Errorf(CodePayload, "wrapping: %w", sentinel)

	var ce *CodedError
	if !errors.As(err, &ce) {
		t.Fatalf("Errorf did not produce a *CodedError: %T", err)
	}
	if ce.Code() != CodePayload {
		t.Errorf("Code() = %d, want %d (%s)", ce.Code(), CodePayload, CodeName(CodePayload))
	}
	if ce.Error() != "wrapping: root cause" {
		t.Errorf("Error() = %q, want %q", ce.Error(), "wrapping: root cause")
	}
	if !errors.Is(err, sentinel) {
		t.Error("errors.Is could not find the wrapped sentinel")
	}
}

// TestClassifyClientError verifies the exit code ClassifyClientError assigns to
// HTTP, invalid-argument, network, and undecodable-body errors, and that it
// keeps the original error in the chain.
func TestClassifyClientError(t *testing.T) {
	var target map[string]any
	syntaxErr := json.Unmarshal([]byte(`{`), &target)
	typeErr := json.Unmarshal([]byte(`[]`), &target)

	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", err: nil, want: CodeSuccess},
		{name: "http", err: fmt.Errorf("request: %w", client.UnsuccessfulHTTPError), want: CodeHTTP},
		{name: "invalid argument", err: fmt.Errorf("GetGroupMembers(): %w: group label cannot be empty", client.InvalidArgumentError), want: CodeUsage},
		{name: "network", err: errors.New("connection refused"), want: CodeNetwork},
		{name: "malformed body", err: fmt.Errorf("failed to unmarshal response: %w", syntaxErr), want: CodePayload},
		{name: "mistyped body", err: fmt.Errorf("failed to unmarshal response: %w", typeErr), want: CodePayload},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyClientError(tc.err, "HTTP failed", "network failed")
			if code := ExitCode(got); code != tc.want {
				t.Errorf("ExitCode() = %d, want %d (%s)", code, tc.want, CodeName(tc.want))
			}
			if tc.err != nil && !errors.Is(got, tc.err) {
				t.Errorf("ClassifyClientError() did not preserve %v", tc.err)
			}
		})
	}
}

// TestAggregateItemErrors verifies that AggregateItemErrors returns nil when no
// item failed, wraps every item error, and codes the result with the code the
// failures share or with CodeMixed when they differ.
func TestAggregateItemErrors(t *testing.T) {
	httpErr := fmt.Errorf("item: %w", client.UnsuccessfulHTTPError)
	netErr := errors.New("connection refused")
	authErr := Errorf(CodeAuth, "token expired")

	tests := []struct {
		name     string
		errs     []error
		wantCode int
	}{
		{name: "no failures", errs: []error{nil, nil}, wantCode: CodeSuccess},
		{name: "all http", errs: []error{nil, httpErr, httpErr}, wantCode: CodeHTTP},
		{name: "all network", errs: []error{netErr, nil, netErr}, wantCode: CodeNetwork},
		{name: "coded item keeps its code", errs: []error{authErr, authErr}, wantCode: CodeAuth},
		{name: "http and network", errs: []error{httpErr, nil, netErr}, wantCode: CodeMixed},
		{name: "coded and http", errs: []error{authErr, httpErr}, wantCode: CodeMixed},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := AggregateItemErrors(zerolog.Nop(), tc.errs, "resource update")
			if code := ExitCode(err); code != tc.wantCode {
				t.Errorf("ExitCode() = %d, want %d (%s)", code, tc.wantCode, CodeName(tc.wantCode))
			}
			if tc.wantCode == CodeSuccess {
				return
			}
			if err.Error() != "resource update completed with errors" {
				t.Errorf("AggregateItemErrors() = %q, want aggregate message", err)
			}
			for _, itemErr := range tc.errs {
				if itemErr != nil && !errors.Is(err, itemErr) {
					t.Errorf("AggregateItemErrors() does not wrap %v", itemErr)
				}
			}
		})
	}
}

// TestCodedError_NilInnerMessage verifies a CodedError with no wrapped error
// still produces a message.
func TestCodedError_NilInnerMessage(t *testing.T) {
	ce := &CodedError{code: CodeGeneric}
	if ce.Error() == "" {
		t.Error("Error() returned empty string for a nil inner error")
	}
	if ce.Unwrap() != nil {
		t.Error("Unwrap() should be nil when there is no inner error")
	}
}

// TestEnsureCode_Nil verifies EnsureCode returns nil for a nil error.
func TestEnsureCode_Nil(t *testing.T) {
	if EnsureCode(CodeHTTP, nil) != nil {
		t.Error("EnsureCode(_, nil) should return nil")
	}
}

// TestEnsureCode_NoDoubleWrap verifies EnsureCode preserves an already-coded
// error's original code rather than overwriting it.
func TestEnsureCode_NoDoubleWrap(t *testing.T) {
	inner := Errorf(CodePayload, "payload problem")
	wrapped := EnsureCode(CodeUsage, inner)

	if ExitCode(wrapped) != CodePayload {
		t.Errorf("ExitCode = %d, want %d (%s); the original code should be preserved", ExitCode(wrapped), CodePayload, CodeName(CodePayload))
	}
}

// TestExitCode_Resolution verifies that ExitCode resolves a nil error to
// CodeSuccess, a coded error (wrapped or not) to its code, the HTTP and config
// sentinels to CodeHTTP and CodeConfig, and any other error to CodeGeneric.
func TestExitCode_Resolution(t *testing.T) {
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
				t.Errorf("ExitCode = %d, want %d (%s)", got, tc.want, CodeName(tc.want))
			}
		})
	}
}

// TestExitCode_CodedBeatsSentinel verifies that an explicit code wins even when
// the wrapped chain also contains a known sentinel.
func TestExitCode_CodedBeatsSentinel(t *testing.T) {
	// A coded error whose inner chain also wraps the HTTP sentinel: the
	// explicit code should take precedence over the sentinel mapping.
	err := Errorf(CodeNetwork, "transport: %w", client.UnsuccessfulHTTPError)
	if got := ExitCode(err); got != CodeNetwork {
		t.Errorf("ExitCode = %d, want %d (%s); the explicit code should win over the sentinel", got, CodeNetwork, CodeName(CodeNetwork))
	}
}

// TestWrapUsageErrors_FlagError verifies that after WrapUsageErrors, a flag
// parsing error resolves to CodeUsage.
func TestWrapUsageErrors_FlagError(t *testing.T) {
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
		t.Errorf("ExitCode = %d, want %d (%s)", ExitCode(err), CodeUsage, CodeName(CodeUsage))
	}
}

// TestWrapUsageErrors_ArgError verifies that after WrapUsageErrors, an
// argument-count violation from a built-in Args validator resolves to
// CodeUsage.
func TestWrapUsageErrors_ArgError(t *testing.T) {
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
		t.Errorf("ExitCode = %d, want %d (%s)", ExitCode(err), CodeUsage, CodeName(CodeUsage))
	}
}

// TestWrapUsageErrors_PreservesExplicitCode verifies that an Args validator
// returning an explicit CodedError keeps its own code instead of being coerced
// to CodeUsage.
func TestWrapUsageErrors_PreservesExplicitCode(t *testing.T) {
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
		t.Errorf("ExitCode = %d, want %d (%s); the explicit code should be preserved", ExitCode(err), CodePayload, CodeName(CodePayload))
	}
}

// TestWrapUsageErrors_Recurses verifies WrapUsageErrors applies to subcommands,
// so an arg error on a child command also resolves to CodeUsage.
func TestWrapUsageErrors_Recurses(t *testing.T) {
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
		t.Errorf("ExitCode = %d, want %d (%s)", ExitCode(err), CodeUsage, CodeName(CodeUsage))
	}
}

// TestCombineItemErrors verifies CombineItemErrors classifies without
// requiring the caller's logger.
func TestCombineItemErrors(t *testing.T) {
	if err := CombineItemErrors(nil, "resource update"); err != nil {
		t.Fatalf("CombineItemErrors(nil) = %v, want nil", err)
	}
	err := CombineItemErrors([]error{errors.New("connection refused")}, "resource update")
	if code := ExitCode(err); code != CodeNetwork {
		t.Errorf("ExitCode() = %d, want %d (%s)", code, CodeNetwork, CodeName(CodeNetwork))
	}
}

// TestCodeName_MatchesConstants verifies CodeName returns each exit code
// constant's own identifier, so failure messages that use it stay accurate
// when a constant is renamed or added.
func TestCodeName_MatchesConstants(t *testing.T) {
	f, err := parser.ParseFile(token.NewFileSet(), "errors.go", nil, 0)
	if err != nil {
		t.Fatalf("failed to parse errors.go: %v", err)
	}
	found := 0
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			vs := spec.(*ast.ValueSpec)
			for i, name := range vs.Names {
				if !strings.HasPrefix(name.Name, "Code") {
					continue
				}
				lit, ok := vs.Values[i].(*ast.BasicLit)
				if !ok {
					t.Fatalf("%s is not an integer literal", name.Name)
				}
				code, err := strconv.Atoi(lit.Value)
				if err != nil {
					t.Fatalf("%s = %s: %v", name.Name, lit.Value, err)
				}
				if got := CodeName(code); got != name.Name {
					t.Errorf("CodeName(%d) = %q, want %q", code, got, name.Name)
				}
				found++
			}
		}
	}
	if found == 0 {
		t.Fatal("no exit code constants found in errors.go")
	}
}

// TestCodeName_Unknown verifies CodeName formats a value outside the exit code
// contract as its number.
func TestCodeName_Unknown(t *testing.T) {
	if got, want := CodeName(42), "Code(42)"; got != want {
		t.Errorf("CodeName(42) = %q, want %q", got, want)
	}
}
