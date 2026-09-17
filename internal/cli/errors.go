// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openchami/ochami/internal/config"
	"github.com/openchami/ochami/pkg/client"
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
)

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

// CodeError wraps an existing error with an explicit exit code without altering
// its message. If err is nil, nil is returned. If err already carries a
// CodedError (anywhere in its chain), it is returned unchanged so the more
// specific, originally-assigned code is preserved.
func CodeError(code int, err error) error {
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
	var (
		eicv config.ErrInvalidConfigVal
		euc  config.ErrUnknownCluster
		emu  config.ErrMissingURI
		eiu  config.ErrInvalidURI
		eisu config.ErrInvalidServiceURI
		eus  config.ErrUnknownService
	)
	return errors.As(err, &eicv) ||
		errors.As(err, &euc) ||
		errors.As(err, &emu) ||
		errors.As(err, &eiu) ||
		errors.As(err, &eisu) ||
		errors.As(err, &eus)
}

// WrapUsageErrors recursively configures cmd and all of its subcommands so that
// usage errors resolve to CodeUsage:
//
//   - Flag parsing errors (e.g. unknown flag, bad flag value) are wrapped via
//     each command's FlagErrorFunc.
//   - Argument-count/validation errors from a command's Args validator (e.g.
//     cobra.ExactArgs, cobra.MinimumNArgs, or a custom validator returning a
//     plain error) are wrapped by composing the existing Args validator.
//
// Explicit CodedErrors returned by a validator are preserved as-is (their code
// wins), so a validator may still return, for example, cli.Errorf(CodePayload,
// ...) if that is more appropriate. This should be called once on the root
// command after the full command tree has been assembled.
func WrapUsageErrors(cmd *cobra.Command) {
	// Wrap flag parse errors as usage errors.
	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		return CodeError(CodeUsage, err)
	})

	// Compose the existing Args validator (if any) so its errors become
	// usage errors. If no validator is set, there is nothing to wrap; Cobra
	// will accept arbitrary args and no arg error can occur.
	if cmd.Args != nil {
		inner := cmd.Args
		cmd.Args = func(c *cobra.Command, args []string) error {
			if err := inner(c, args); err != nil {
				return CodeError(CodeUsage, err)
			}
			return nil
		}
	}

	// Recurse into subcommands.
	for _, sub := range cmd.Commands() {
		WrapUsageErrors(sub)
	}
}
