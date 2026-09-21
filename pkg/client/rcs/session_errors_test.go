// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rcs

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestEnableRawTerminalMode_Error verifies that enableRawTerminalMode returns
// the terminal controller's failure to enter raw mode.
func TestEnableRawTerminalMode_Error(t *testing.T) {
	file, err := os.Create(filepath.Join(t.TempDir(), "stdin"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	want := errors.New("raw unavailable")
	if _, err := enableRawTerminalMode(file, &fakeTerminal{makeErr: want}); !errors.Is(err, want) {
		t.Fatalf("enableRawTerminalMode() error = %v", err)
	}
}

// TestRunConsoleSession_TerminalErrors verifies that runConsoleSession returns
// a failure to put the terminal into raw mode, and that on exit it restores the
// terminal and joins a restore failure with a close failure.
func TestRunConsoleSession_TerminalErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stdin")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	t.Run("setup failure", func(t *testing.T) {
		want := errors.New("raw setup failed")
		conn := &fakeMessageConn{}
		err := runConsoleSession(context.Background(), conn, &fakeTerminal{isTerminal: true, makeErr: want}, make(chan os.Signal), time.After, file, io.Discard)
		if !errors.Is(err, want) {
			t.Fatalf("runConsoleSession() error = %v, want setup error", err)
		}
	})

	t.Run("joins restore and close failures", func(t *testing.T) {
		restoreErr := errors.New("restore failed")
		closeErr := errors.New("close failed")
		conn := &fakeMessageConn{
			readErr:  &websocket.CloseError{Code: websocket.CloseNormalClosure},
			closeErr: closeErr,
		}
		terminal := &fakeTerminal{isTerminal: true, restoreErr: restoreErr}
		err := runConsoleSession(context.Background(), conn, terminal, make(chan os.Signal), time.After, file, io.Discard)
		if !errors.Is(err, restoreErr) || !errors.Is(err, closeErr) {
			t.Fatalf("runConsoleSession() error = %v, want restore and close errors", err)
		}
		if !terminal.restored {
			t.Fatal("terminal was not restored")
		}
	})
}

// TestShowConsole_ReturnsConnectionCloseError verifies that ShowConsole returns
// a failure to close the websocket connection after a normal close from the
// server.
func TestShowConsole_ReturnsConnectionCloseError(t *testing.T) {
	want := errors.New("close failed")
	conn := &fakeMessageConn{
		readErr:  &websocket.CloseError{Code: websocket.CloseNormalClosure},
		closeErr: want,
	}
	c, err := NewClient("https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	c.dial = func(context.Context, string, http.Header) (messageConn, *http.Response, error) {
		return conn, nil, nil
	}
	if err := c.ShowConsole(context.Background(), "x0", false, 1, "", io.Discard); !errors.Is(err, want) {
		t.Fatalf("ShowConsole() error = %v, want close error", err)
	}
}
