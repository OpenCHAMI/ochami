// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// root_test.go exercises handleExecuteError, the testable core of Execute that
// resolves an error to a process exit code and emits the help hint, without
// terminating the test binary via os.Exit.

import (
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestHandleExecuteError verifies that handleExecuteError maps a nil error to
// CodeSuccess and a coded error (HTTP, generic, or declined) to its code.
func TestHandleExecuteError(t *testing.T) {
	rootCmd := NewRootCmd()

	// nil error -> success.
	if code := handleExecuteError(rootCmd, nil); code != cli.CodeSuccess {
		t.Errorf("handleExecuteError(nil) = %d, want %d (%s)", code, cli.CodeSuccess, cli.CodeName(cli.CodeSuccess))
	}

	// A coded error resolves to its code and emits the help hint.
	err := cli.Errorf(cli.CodeHTTP, "boom")
	if code := handleExecuteError(rootCmd, err); code != cli.CodeHTTP {
		t.Errorf("handleExecuteError(%s err) = %d, want %d", cli.CodeName(cli.CodeHTTP), code, cli.CodeHTTP)
	}

	// A plain error resolves to the generic code.
	plain := cli.Errorf(cli.CodeGeneric, "plain")
	if code := handleExecuteError(rootCmd, plain); code != cli.CodeGeneric {
		t.Errorf("handleExecuteError(generic err) = %d, want %d (%s)", code, cli.CodeGeneric, cli.CodeName(cli.CodeGeneric))
	}

	// A declined prompt resolves to CodeDeclined.
	declined := cli.Errorf(cli.CodeDeclined, "user declined")
	if code := handleExecuteError(rootCmd, declined); code != cli.CodeDeclined {
		t.Errorf("handleExecuteError(declined err) = %d, want %d (%s)", code, cli.CodeDeclined, cli.CodeName(cli.CodeDeclined))
	}
}
