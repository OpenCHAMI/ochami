// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rcs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

type fakeTerminal struct {
	isTerminal bool
	makeErr    error
	restoreErr error
	madeRaw    bool
	restored   bool
}

func (f *fakeTerminal) IsTerminal(int) bool { return f.isTerminal }
func (f *fakeTerminal) MakeRaw(int) (*term.State, error) {
	f.madeRaw = true
	return &term.State{}, f.makeErr
}
func (f *fakeTerminal) Restore(int, *term.State) error {
	f.restored = true
	return f.restoreErr
}

type fakeMessageConn struct {
	readType int
	readData []byte
	readErr  error
	nextErr  error
	reads    int
	writeErr error
	closeErr error
	writes   [][]byte
}

func (f *fakeMessageConn) ReadMessage() (int, []byte, error) {
	f.reads++
	if f.reads > 1 && f.nextErr != nil {
		return 0, nil, f.nextErr
	}
	return f.readType, f.readData, f.readErr
}
func (f *fakeMessageConn) WriteMessage(_ int, data []byte) error {
	f.writes = append(f.writes, append([]byte(nil), data...))
	return f.writeErr
}
func (f *fakeMessageConn) Close() error { return f.closeErr }

func TestTerminalLifecycleUsesInjectedController(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stdin")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	controller := &fakeTerminal{isTerminal: true}
	gotFile, ok := terminalInputFile(file, controller)
	if !ok || gotFile != file {
		t.Fatal("terminalInputFile() did not recognize injected terminal")
	}
	state, err := enableRawTerminalMode(file, controller)
	if err != nil || state == nil || !controller.madeRaw {
		t.Fatalf("enableRawTerminalMode() = (%v, %v), madeRaw=%v", state, err, controller.madeRaw)
	}
	restore := terminalInputState{file: file, state: state, controller: controller}
	if err := restore.Restore(); err != nil || !controller.restored {
		t.Fatalf("Restore() = %v, restored=%v", err, controller.restored)
	}
}

func TestEnableRawTerminalModeError(t *testing.T) {
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

func TestWaitForConsoleExit(t *testing.T) {
	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := waitForConsoleExit(ctx, &fakeMessageConn{}, make(chan os.Signal), make(chan struct{}), make(chan error))
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("waitForConsoleExit() = %v, want context.Canceled", err)
		}
	})

	t.Run("stream error", func(t *testing.T) {
		want := errors.New("stream failed")
		errs := make(chan error, 1)
		errs <- want
		if err := waitForConsoleExit(context.Background(), &fakeMessageConn{}, make(chan os.Signal), make(chan struct{}), errs); !errors.Is(err, want) {
			t.Fatalf("waitForConsoleExit() = %v, want stream error", err)
		}
	})

	t.Run("close message failure", func(t *testing.T) {
		want := errors.New("write failed")
		interrupt := make(chan os.Signal, 1)
		interrupt <- syscall.SIGINT
		err := waitForConsoleExit(context.Background(), &fakeMessageConn{writeErr: want}, interrupt, make(chan struct{}), make(chan error))
		if !errors.Is(err, want) {
			t.Fatalf("waitForConsoleExit() = %v, want close write error", err)
		}
	})

	t.Run("clean interrupt", func(t *testing.T) {
		interrupt := make(chan os.Signal, 1)
		interrupt <- syscall.SIGINT
		done := make(chan struct{})
		close(done)
		conn := &fakeMessageConn{}
		if err := waitForConsoleExit(context.Background(), conn, interrupt, done, make(chan error)); err != nil {
			t.Fatalf("waitForConsoleExit() = %v", err)
		}
		if len(conn.writes) != 1 {
			t.Fatalf("close messages = %d, want 1", len(conn.writes))
		}
	})
}

func TestStreamConsoleOutput(t *testing.T) {
	t.Run("writes supported message", func(t *testing.T) {
		conn := &fakeMessageConn{readType: websocket.TextMessage, readData: []byte("hello"), nextErr: io.EOF}
		var out bytes.Buffer
		errs := make(chan error, 1)
		done := make(chan struct{})
		go streamConsoleOutput(&out, conn, errs, done)
		select {
		case err := <-errs:
			if !errors.Is(err, io.EOF) {
				t.Fatalf("stream error = %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("streamConsoleOutput did not finish")
		}
		if out.String() != "hello" {
			t.Errorf("output = %q, want hello", out.String())
		}
	})
}

func TestShowConsoleReturnsConnectionCloseError(t *testing.T) {
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
