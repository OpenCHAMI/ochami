// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rcs

// input_test.go unit-tests the console input-forwarding helpers in isolation
// using a fake messageWriter (no live websocket). It covers keystroke
// forwarding, the raw-mode Ctrl+C (ETX) -> SIGINT translation, and the buffered
// input path.

import (
	"errors"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// fakeMessageWriter records messages written to it and can be configured to
// return an error on write.
type fakeMessageWriter struct {
	mu       sync.Mutex
	messages [][]byte
	err      error
}

func (f *fakeMessageWriter) WriteMessage(_ int, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	// Copy since the caller reuses the buffer.
	cp := make([]byte, len(data))
	copy(cp, data)
	f.messages = append(f.messages, cp)
	return nil
}

func (f *fakeMessageWriter) written() [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.messages
}

// TestStreamBufferedConsoleInput verifies buffered input is forwarded to the
// writer and that reaching EOF ends the stream without reporting an error.
func TestStreamBufferedConsoleInput(t *testing.T) {
	fw := &fakeMessageWriter{}
	errChan := make(chan error, 1)

	done := make(chan struct{})
	go func() {
		streamBufferedConsoleInput(strings.NewReader("uptime\n"), fw, errChan)
		close(done)
	}()

	select {
	case <-done:
	case err := <-errChan:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("streamBufferedConsoleInput did not return on EOF")
	}

	got := flatten(fw.written())
	if !strings.Contains(got, "uptime") {
		t.Errorf("forwarded = %q, want it to contain the input", got)
	}
}

// TestStreamRawConsoleInputForwards verifies raw-mode input is forwarded
// byte-by-byte to the writer.
func TestStreamRawConsoleInputForwards(t *testing.T) {
	fw := &fakeMessageWriter{}
	interrupt := make(chan os.Signal, 1)
	errChan := make(chan error, 1)

	done := make(chan struct{})
	go func() {
		streamRawConsoleInput(strings.NewReader("ls"), fw, interrupt, errChan)
		close(done)
	}()

	select {
	case <-done:
	case err := <-errChan:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("streamRawConsoleInput did not return on EOF")
	}

	got := flatten(fw.written())
	if got != "ls" {
		t.Errorf("forwarded = %q, want %q", got, "ls")
	}
}

// TestStreamRawConsoleInputCtrlC verifies that a Ctrl+C (ETX) byte in raw mode
// is translated into a SIGINT on the interrupt channel and stops forwarding.
func TestStreamRawConsoleInputCtrlC(t *testing.T) {
	fw := &fakeMessageWriter{}
	interrupt := make(chan os.Signal, 1)
	errChan := make(chan error, 1)

	// "a" then Ctrl+C then "b": only "a" should be forwarded, and an interrupt
	// should be signaled before "b" is read.
	input := string([]byte{'a', ctrlCByte, 'b'})
	go streamRawConsoleInput(strings.NewReader(input), fw, interrupt, errChan)

	select {
	case sig := <-interrupt:
		if sig != syscall.SIGINT {
			t.Errorf("interrupt signal = %v, want SIGINT", sig)
		}
	case err := <-errChan:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("expected an interrupt signal, got none")
	}

	got := flatten(fw.written())
	if got != "a" {
		t.Errorf("forwarded = %q, want only %q before Ctrl+C", got, "a")
	}
}

// TestStreamRawConsoleInputWriteError verifies a write error is surfaced on the
// error channel and stops forwarding.
func TestStreamRawConsoleInputWriteError(t *testing.T) {
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

// TestTerminalInputFileNonFile verifies terminalInputFile reports false for a
// reader that is not an *os.File (e.g. an in-memory reader), so buffered mode is
// used for non-terminal input.
func TestTerminalInputFileNonFile(t *testing.T) {
	if _, ok := terminalInputFile(strings.NewReader("x"), systemTerminal{}); ok {
		t.Error("terminalInputFile reported true for a non-file reader")
	}
}

// flatten concatenates recorded messages into a single string.
func flatten(msgs [][]byte) string {
	var b strings.Builder
	for _, m := range msgs {
		b.Write(m)
	}
	return b.String()
}
