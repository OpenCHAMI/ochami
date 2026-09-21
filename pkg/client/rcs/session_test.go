// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rcs

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	wrote    chan struct{}
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
	if f.wrote != nil {
		f.wrote <- struct{}{}
	}
	return f.writeErr
}

func (f *fakeMessageConn) Close() error { return f.closeErr }

type zeroThenReader struct {
	reads int
}

func (r *zeroThenReader) Read(data []byte) (int, error) {
	r.reads++
	switch r.reads {
	case 1:
		return 0, nil
	case 2:
		data[0] = 'x'
		return 1, nil
	default:
		return 0, io.EOF
	}
}

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) { return 0, r.err }

// TestTerminalLifecycle_UsesInjectedController verifies that terminal
// detection, entering raw mode, and restoring the terminal all go through the
// injected terminal controller.
func TestTerminalLifecycle_UsesInjectedController(t *testing.T) {
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

// TestWaitForConsoleExit verifies that waitForConsoleExit returns a context
// cancellation or a stream error, returns a failure to send the close frame
// after an interrupt, and otherwise exits cleanly after an interrupt, even when
// the read side reports an abnormal close or doesn't finish within the grace
// period.
func TestWaitForConsoleExit(t *testing.T) {
	neverAfter := func(time.Duration) <-chan time.Time { return make(chan time.Time) }
	newChannels := func() (chan os.Signal, chan struct{}, chan error, chan error) {
		return make(chan os.Signal), make(chan struct{}), make(chan error), make(chan error)
	}

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		interrupt, localInterrupt, inputErr, outputErr := newChannels()
		err := waitForConsoleExit(ctx, &fakeMessageConn{}, interrupt, localInterrupt, inputErr, outputErr, neverAfter)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("waitForConsoleExit() = %v, want context.Canceled", err)
		}
	})

	t.Run("stream error", func(t *testing.T) {
		want := errors.New("stream failed")
		interrupt, localInterrupt, _, outputErr := newChannels()
		inputErr := make(chan error, 1)
		inputErr <- want
		if err := waitForConsoleExit(context.Background(), &fakeMessageConn{}, interrupt, localInterrupt, inputErr, outputErr, neverAfter); !errors.Is(err, want) {
			t.Fatalf("waitForConsoleExit() = %v, want stream error", err)
		}
	})

	t.Run("close message failure", func(t *testing.T) {
		want := errors.New("write failed")
		interrupt := make(chan os.Signal, 1)
		interrupt <- syscall.SIGINT
		err := waitForConsoleExit(context.Background(), &fakeMessageConn{writeErr: want}, interrupt, make(chan struct{}), make(chan error), make(chan error), neverAfter)
		if !errors.Is(err, want) {
			t.Fatalf("waitForConsoleExit() = %v, want close write error", err)
		}
	})

	t.Run("clean interrupt", func(t *testing.T) {
		interrupt := make(chan os.Signal, 1)
		interrupt <- syscall.SIGINT
		outputErr := make(chan error, 1)
		conn := &fakeMessageConn{wrote: make(chan struct{}, 1)}
		go func() {
			<-conn.wrote
			outputErr <- nil
		}()
		if err := waitForConsoleExit(context.Background(), conn, interrupt, make(chan struct{}), make(chan error), outputErr, neverAfter); err != nil {
			t.Fatalf("waitForConsoleExit() = %v", err)
		}
		if len(conn.writes) != 1 {
			t.Fatalf("close messages = %d, want 1", len(conn.writes))
		}
	})

	t.Run("interrupt with abnormal output error is not surfaced", func(t *testing.T) {
		// A user-initiated interrupt must exit cleanly even if the remote
		// tears down the connection abnormally (no clean close handshake)
		// within the grace window: the interrupt, not the read side, decides
		// the outcome.
		interrupt := make(chan os.Signal, 1)
		interrupt <- syscall.SIGINT
		outputErr := make(chan error, 1)
		conn := &fakeMessageConn{wrote: make(chan struct{}, 1)}
		go func() {
			<-conn.wrote
			outputErr <- &websocket.CloseError{Code: websocket.CloseInternalServerErr}
		}()
		if err := waitForConsoleExit(context.Background(), conn, interrupt, make(chan struct{}), make(chan error), outputErr, neverAfter); err != nil {
			t.Fatalf("waitForConsoleExit() = %v, want nil despite abnormal output error", err)
		}
	})

	t.Run("interrupt timeout", func(t *testing.T) {
		localInterrupt := make(chan struct{}, 1)
		localInterrupt <- struct{}{}
		timedOut := make(chan time.Time)
		close(timedOut)
		conn := &fakeMessageConn{}
		if err := waitForConsoleExit(context.Background(), conn, make(chan os.Signal), localInterrupt, make(chan error), make(chan error), func(time.Duration) <-chan time.Time { return timedOut }); err != nil {
			t.Fatalf("waitForConsoleExit() = %v", err)
		}
		if len(conn.writes) != 1 {
			t.Fatalf("close messages = %d, want 1", len(conn.writes))
		}
	})
}

// TestStreamConsoleOutput verifies that streamConsoleOutput writes received
// text and binary messages to the output, skips other message types, and
// reports the read error that ends the stream.
func TestStreamConsoleOutput(t *testing.T) {
	t.Run("writes supported message", func(t *testing.T) {
		conn := &fakeMessageConn{readType: websocket.TextMessage, readData: []byte("hello"), nextErr: io.EOF}
		var out bytes.Buffer
		errs := make(chan error, 1)
		go streamConsoleOutput(&out, conn, errs)
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

	t.Run("writes binary message", func(t *testing.T) {
		conn := &fakeMessageConn{readType: websocket.BinaryMessage, readData: []byte{1, 2}, nextErr: io.EOF}
		var out bytes.Buffer
		errs := make(chan error, 1)
		go streamConsoleOutput(&out, conn, errs)
		if err := <-errs; !errors.Is(err, io.EOF) {
			t.Fatalf("stream error = %v, want io.EOF", err)
		}
		if !bytes.Equal(out.Bytes(), []byte{1, 2}) {
			t.Errorf("output = %v, want [1 2]", out.Bytes())
		}
	})

	t.Run("ignores unsupported message", func(t *testing.T) {
		conn := &fakeMessageConn{readType: websocket.PingMessage, readData: []byte("ignored"), nextErr: io.EOF}
		var out bytes.Buffer
		errs := make(chan error, 1)
		go streamConsoleOutput(&out, conn, errs)
		if err := <-errs; !errors.Is(err, io.EOF) {
			t.Fatalf("stream error = %v, want io.EOF", err)
		}
		if out.Len() != 0 {
			t.Errorf("output = %q, want empty", out.String())
		}
	})

	t.Run("normal close", func(t *testing.T) {
		errs := make(chan error, 1)
		go streamConsoleOutput(io.Discard, &fakeMessageConn{readErr: &websocket.CloseError{Code: websocket.CloseNormalClosure}}, errs)
		if err := <-errs; err != nil {
			t.Fatalf("stream error = %v, want nil", err)
		}
	})

	t.Run("abnormal close", func(t *testing.T) {
		want := &websocket.CloseError{Code: websocket.CloseInternalServerErr}
		errs := make(chan error, 1)
		go streamConsoleOutput(io.Discard, &fakeMessageConn{readErr: want}, errs)
		if err := <-errs; !errors.Is(err, want) {
			t.Fatalf("stream error = %v, want abnormal close", err)
		}
	})
}

// TestConsoleInput_FailuresAndZeroByteReads verifies that both console input
// paths skip a zero-byte read and keep forwarding, and report read errors and
// (for the buffered path) websocket write errors on the error channel.
func TestConsoleInput_FailuresAndZeroByteReads(t *testing.T) {
	t.Run("buffered zero-byte read", func(t *testing.T) {
		reader := &zeroThenReader{}
		writer := &fakeMessageWriter{}
		streamBufferedConsoleInput(reader, writer, make(chan error, 1))
		if got := flatten(writer.written()); got != "x" {
			t.Fatalf("forwarded input = %q, want x", got)
		}
	})

	t.Run("raw zero-byte read", func(t *testing.T) {
		reader := &zeroThenReader{}
		writer := &fakeMessageWriter{}
		streamRawConsoleInput(reader, writer, make(chan struct{}, 1), make(chan error, 1))
		if got := flatten(writer.written()); got != "x" {
			t.Fatalf("forwarded input = %q, want x", got)
		}
	})

	for _, mode := range []string{"buffered", "raw"} {
		t.Run(mode+" read error", func(t *testing.T) {
			want := errors.New("read failed")
			errs := make(chan error, 1)
			if mode == "buffered" {
				streamBufferedConsoleInput(errorReader{err: want}, &fakeMessageWriter{}, errs)
			} else {
				streamRawConsoleInput(errorReader{err: want}, &fakeMessageWriter{}, make(chan struct{}, 1), errs)
			}
			if err := <-errs; !errors.Is(err, want) {
				t.Fatalf("input error = %v, want read error", err)
			}
		})
	}

	t.Run("buffered websocket write error", func(t *testing.T) {
		want := errors.New("write failed")
		errs := make(chan error, 1)
		streamBufferedConsoleInput(strings.NewReader("x"), &fakeMessageWriter{err: want}, errs)
		if err := <-errs; !errors.Is(err, want) {
			t.Fatalf("input error = %v, want websocket write error", err)
		}
	})
}

type shortWriter struct {
	err error
}

func (w shortWriter) Write(data []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	return len(data) - 1, nil
}

// TestWriteOutput verifies that writeOutput reports a short write as
// io.ErrShortWrite and returns the writer's own error unchanged.
func TestWriteOutput(t *testing.T) {
	if err := writeOutput(shortWriter{}, []byte("console")); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("writeOutput() error = %v, want io.ErrShortWrite", err)
	}
	want := errors.New("output failed")
	if err := writeOutput(shortWriter{err: want}, []byte("console")); !errors.Is(err, want) {
		t.Fatalf("writeOutput() error = %v, want output error", err)
	}
}

// TestRunConsoleSession_ImmediateCancellation verifies that an already-canceled
// context ends the session before any websocket read, and that the returned
// error joins the cancellation with the connection's close failure.
func TestRunConsoleSession_ImmediateCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	closeErr := errors.New("close failed")
	conn := &fakeMessageConn{closeErr: closeErr}
	err := runConsoleSession(ctx, conn, &fakeTerminal{}, make(chan os.Signal), time.After, strings.NewReader(""), io.Discard)
	if !errors.Is(err, context.Canceled) || !errors.Is(err, closeErr) {
		t.Fatalf("runConsoleSession() error = %v, want canceled and close errors", err)
	}
	if conn.reads != 0 {
		t.Fatalf("websocket reads = %d, want 0", conn.reads)
	}
}
