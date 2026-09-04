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

func TestHandleExecuteError(t *testing.T) {
	rootCmd := NewRootCmd()

	// nil error -> success.
	if code := handleExecuteError(rootCmd, nil); code != cli.CodeSuccess {
		t.Errorf("handleExecuteError(nil) = %d, want %d", code, cli.CodeSuccess)
	}

	// A coded error resolves to its code and emits the help hint.
	err := cli.Errorf(cli.CodeHTTP, "boom")
	if code := handleExecuteError(rootCmd, err); code != cli.CodeHTTP {
		t.Errorf("handleExecuteError(CodeHTTP err) = %d, want %d", code, cli.CodeHTTP)
	}

	// A plain error resolves to the generic code.
	plain := cli.Errorf(cli.CodeGeneric, "plain")
	if code := handleExecuteError(rootCmd, plain); code != cli.CodeGeneric {
		t.Errorf("handleExecuteError(generic err) = %d, want %d", code, cli.CodeGeneric)
	}
}
