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
