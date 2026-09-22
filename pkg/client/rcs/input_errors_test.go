// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rcs

// input_errors_test.go covers the write-error rejection path for the console
// input-forwarding helpers; see input_test.go for the success paths these
// mirror.

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// TestStreamRawConsoleInput_WriteError verifies a write error is surfaced on the
// error channel and stops forwarding.
func TestStreamRawConsoleInput_WriteError(t *testing.T) {
	fw := &fakeMessageWriter{err: errors.New("write failed")}
	interrupt := make(chan os.Signal, 1)
	errChan := make(chan error, 1)

	go streamRawConsoleInput(strings.NewReader("x"), fw, interrupt, errChan)

	select {
	case err := <-errChan:
		if err == nil {
			t.Fatal("expected a write error, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected a write error on the error channel, got none")
	}
}
