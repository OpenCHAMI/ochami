// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package rcs

// input_test.go unit-tests the console input-forwarding helpers in isolation
// using a fake messageWriter (no live websocket). It covers keystroke
// forwarding, the raw-mode Ctrl+C (ETX) -> SIGINT translation, and the buffered
// input path. The write-error rejection path is covered in
// input_errors_test.go.

import (
	"io"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// eofWithDataReader returns its entire payload together with io.EOF in a
// single Read call. io.Reader permits this (a read may return n > 0 and a
// non-nil error, including io.EOF, in the same call), and some real readers
// (pipes, files, sockets) do it on stream close.
type eofWithDataReader struct {
	data []byte
	done bool
}

func (r *eofWithDataReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, io.EOF
	}
	r.done = true
	n := copy(p, r.data)
	return n, io.EOF
}

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

// TestStreamRawConsoleInput_Forwards verifies raw-mode input is forwarded
// byte-by-byte to the writer.
func TestStreamRawConsoleInput_Forwards(t *testing.T) {
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

// TestStreamRawConsoleInput_CtrlC verifies that a Ctrl+C (ETX) byte in raw mode
// is translated into a SIGINT on the interrupt channel and stops forwarding.
func TestStreamRawConsoleInput_CtrlC(t *testing.T) {
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

// TestStreamRawConsoleInput_DataWithEOF verifies a byte read together with
// io.EOF in the same Read call is still forwarded before the stream ends.
func TestStreamRawConsoleInput_DataWithEOF(t *testing.T) {
	fw := &fakeMessageWriter{}
	interrupt := make(chan os.Signal, 1)
	errChan := make(chan error, 1)

	done := make(chan struct{})
	go func() {
		streamRawConsoleInput(&eofWithDataReader{data: []byte("z")}, fw, interrupt, errChan)
		close(done)
	}()

	select {
	case <-done:
	case err := <-errChan:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("streamRawConsoleInput did not return")
	}

	got := flatten(fw.written())
	if got != "z" {
		t.Errorf("forwarded = %q, want %q (byte delivered alongside EOF must not be dropped)", got, "z")
	}
}

// TestStreamBufferedConsoleInput_DataWithEOF verifies buffered bytes read
// together with io.EOF in the same Read call are still forwarded before the
// stream ends.
func TestStreamBufferedConsoleInput_DataWithEOF(t *testing.T) {
	fw := &fakeMessageWriter{}
	errChan := make(chan error, 1)

	done := make(chan struct{})
	go func() {
		streamBufferedConsoleInput(&eofWithDataReader{data: []byte("uptime\n")}, fw, errChan)
		close(done)
	}()

	select {
	case <-done:
	case err := <-errChan:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("streamBufferedConsoleInput did not return")
	}

	got := flatten(fw.written())
	if got != "uptime\n" {
		t.Errorf("forwarded = %q, want %q (bytes delivered alongside EOF must not be dropped)", got, "uptime\n")
	}
}

// TestTerminalInputFile_NonFile verifies terminalInputFile reports false for a
// reader that is not an *os.File (e.g. an in-memory reader), so buffered mode is
// used for non-terminal input.
func TestTerminalInputFile_NonFile(t *testing.T) {
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
