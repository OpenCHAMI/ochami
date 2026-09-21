// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/pkg/client"
	"github.com/openchami/ochami/pkg/config"
)

// Exit code contract for the ochami CLI. These values are a stable, public
// contract: scripts and tooling may rely on them, so they must not be changed
// without a compatibility consideration. Commands generally return a plain
// error and inherit the correct code via ExitCode's sentinel mapping; a command
// may return a CodedError (via Errorf) to force a specific code.
const (
	// CodeSuccess indicates the command completed without error.
	CodeSuccess = 0
	// CodeGeneric is the fallback code for any error without a more specific
	// classification.
	CodeGeneric = 1
	// CodeUsage indicates invalid usage: bad flags, arguments, or mutually
	// exclusive options.
	CodeUsage = 2
	// CodeConfig indicates a configuration error (reading, parsing, or
	// resolving configuration values).
	CodeConfig = 3
	// CodeAuth indicates an authentication/token error (missing, expired, or
	// otherwise invalid token).
	CodeAuth = 4
	// CodePayload indicates a payload, (un)marshalling, input, or output
	// formatting error.
	CodePayload = 5
	// CodeHTTP indicates the server returned an unsuccessful HTTP response.
	CodeHTTP = 6
	// CodeNetwork indicates a network/transport error reaching a service.
	CodeNetwork = 7
	// CodeDeclined indicates the user declined a confirmation prompt (for
	// example, "Really delete?" or an offer to create a missing config file),
	// so the command made no changes.
	CodeDeclined = 8
)

// CodeName returns the identifier of the exit code constant whose value is
// code (for example, "CodeNetwork" for 7), or "Code(<code>)" for a value
// outside the contract. Diagnostics such as test failure messages use it
// instead of hard-coding a constant's name, so they stay accurate if the
// constant is renamed.
func CodeName(code int) string {
	switch code {
	case CodeSuccess:
		return "CodeSuccess"
	case CodeGeneric:
		return "CodeGeneric"
	case CodeUsage:
		return "CodeUsage"
	case CodeConfig:
		return "CodeConfig"
	case CodeAuth:
		return "CodeAuth"
	case CodePayload:
		return "CodePayload"
	case CodeHTTP:
		return "CodeHTTP"
	case CodeNetwork:
		return "CodeNetwork"
	case CodeDeclined:
		return "CodeDeclined"
	default:
		return fmt.Sprintf("Code(%d)", code)
	}
}

// CodedError wraps an error with an associated process exit code so that
// Execute can translate command failures into differentiated exit statuses.
type CodedError struct {
	code int
	err  error
}

// Error implements the error interface.
func (ce *CodedError) Error() string {
	if ce.err == nil {
		return fmt.Sprintf("error (exit code %d)", ce.code)
	}
	return ce.err.Error()
}

// Unwrap allows errors.Is/errors.As to inspect the wrapped error.
func (ce *CodedError) Unwrap() error {
	return ce.err
}

// Code returns the explicit exit code associated with the error.
func (ce *CodedError) Code() int {
	return ce.code
}

// Errorf builds a CodedError with the given exit code and a formatted message.
// It supports %w wrapping, so sentinel errors remain inspectable via
// errors.Is/errors.As.
func Errorf(code int, format string, args ...any) error {
	return &CodedError{code: code, err: fmt.Errorf(format, args...)}
}

// EnsureCode wraps an existing error with an explicit exit code without altering
// its message. If err is nil, nil is returned. If err already carries a
// CodedError (anywhere in its chain), it is returned unchanged so the more
// specific, originally-assigned code is preserved.
func EnsureCode(code int, err error) error {
	if err == nil {
		return nil
	}
	var ce *CodedError
	if errors.As(err, &ce) {
		return err
	}
	return &CodedError{code: code, err: err}
}

// ExitCode resolves the process exit code for an error returned from a command.
// Resolution order:
//  1. nil error -> CodeSuccess.
//  2. An explicit CodedError code (nearest wrapping CodedError wins).
//  3. Known sentinels: unsuccessful HTTP response -> CodeHTTP; configuration
//     error types -> CodeConfig.
//  4. Fallback -> CodeGeneric.
func ExitCode(err error) int {
	if err == nil {
		return CodeSuccess
	}

	// Honor an explicit coded error first.
	var ce *CodedError
	if errors.As(err, &ce) {
		return ce.code
	}

	// Map known sentinels.
	if errors.Is(err, client.UnsuccessfulHTTPError) {
		return CodeHTTP
	}
	if isConfigError(err) {
		return CodeConfig
	}

	return CodeGeneric
}

// isConfigError reports whether err is (or wraps) one of the config package's
// typed errors.
func isConfigError(err error) bool {
	var ce config.Error
	return errors.As(err, &ce)
}

// WrapUsageErrors recursively configures cmd and all of its subcommands so that
// usage errors resolve to CodeUsage:
//
//   - Flag parsing errors (e.g. unknown flag, bad flag value) are wrapped via
//     each command's FlagErrorFunc.
//   - Argument-count/validation errors from a command's Args validator (e.g.
//     cobra.ExactArgs, cobra.MinimumNArgs, or a custom validator returning a
//     plain error) are wrapped by composing the existing Args validator.
//   - Required-flag and flag-group errors (MarkFlagRequired,
//     MarkFlagsMutuallyExclusive, MarkFlagsOneRequired,
//     MarkFlagsRequiredTogether) are wrapped by composing the command's
//     PreRunE/PreRun and re-running Cobra's own
//     ValidateRequiredFlags/ValidateFlagGroups checks immediately afterward.
//     Cobra normally runs these checks itself right after PreRunE, with no
//     hook to customize the resulting error, so this wrapper runs the same
//     checks early (after any existing PreRunE/PreRun, so hooks that
//     register flag groups at runtime still take effect) and returns a
//     wrapped error before Cobra's own unwrapped check is reached.
//
// Explicit CodedErrors returned by a validator are preserved as-is (their code
// wins), so a validator may still return, for example, cli.Errorf(CodePayload,
// ...) if that is more appropriate. This should be called once on the root
// command after the full command tree has been assembled.
func WrapUsageErrors(cmd *cobra.Command) {
	// Wrap flag parse errors as usage errors. Cobra's FlagErrorFunc() getter
	// already falls back to the nearest ancestor's function when a command
	// has none set of its own, so this only needs to be set once, on the
	// true root, rather than on every node in the tree.
	if !cmd.HasParent() {
		cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
			return EnsureCode(CodeUsage, err)
		})
	}

	// Compose the existing Args validator (if any) so its errors become
	// usage errors. If no validator is set, there is nothing to wrap; Cobra
	// will accept arbitrary args and no arg error can occur.
	if cmd.Args != nil {
		inner := cmd.Args
		cmd.Args = func(c *cobra.Command, args []string) error {
			if err := inner(c, args); err != nil {
				return EnsureCode(CodeUsage, err)
			}
			return nil
		}
	}

	// Compose the existing PreRunE/PreRun (if any) so that Cobra's
	// required-flag and flag-group validation, which runs immediately after
	// this hook, resolves to CodeUsage.
	switch {
	case cmd.PreRunE != nil:
		inner := cmd.PreRunE
		cmd.PreRunE = func(c *cobra.Command, args []string) error {
			if err := inner(c, args); err != nil {
				return err
			}
			return validateFlagsAsUsage(c)
		}
	case cmd.PreRun != nil:
		inner := cmd.PreRun
		cmd.PreRun = nil
		cmd.PreRunE = func(c *cobra.Command, args []string) error {
			inner(c, args)
			return validateFlagsAsUsage(c)
		}
	default:
		cmd.PreRunE = func(c *cobra.Command, args []string) error {
			return validateFlagsAsUsage(c)
		}
	}

	// Recurse into subcommands.
	for _, sub := range cmd.Commands() {
		WrapUsageErrors(sub)
	}
}

// validateFlagsAsUsage runs cmd's required-flag and flag-group validation and
// wraps any failure as CodeUsage. Both checks are no-ops if cmd has no
// required flags or flag groups registered.
func validateFlagsAsUsage(cmd *cobra.Command) error {
	if err := cmd.ValidateRequiredFlags(); err != nil {
		return EnsureCode(CodeUsage, err)
	}
	if err := cmd.ValidateFlagGroups(); err != nil {
		return EnsureCode(CodeUsage, err)
	}
	return nil
}
