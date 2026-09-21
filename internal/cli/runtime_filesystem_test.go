// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fileCreationOperationsStub struct {
	stat     func(string) error
	mkdirAll func(string, os.FileMode) error
	openFile func(string, int, os.FileMode) (io.Closer, error)
}

func (s fileCreationOperationsStub) Stat(path string) error {
	return s.stat(path)
}

func (s fileCreationOperationsStub) MkdirAll(path string, mode os.FileMode) error {
	return s.mkdirAll(path, mode)
}

func (s fileCreationOperationsStub) OpenFile(path string, flag int, mode os.FileMode) (io.Closer, error) {
	return s.openFile(path, flag, mode)
}

type closeFunc func() error

func (f closeFunc) Close() error { return f() }

type readError struct{ err error }

func (r readError) Read([]byte) (int, error) { return 0, r.err }

type writeError struct{ err error }

func (w writeError) Write([]byte) (int, error) { return 0, w.err }

func TestIOStreamsConfirmCreate(t *testing.T) {
	t.Parallel()

	readErr := errors.New("read failed")
	writeErr := errors.New("write failed")
	tests := []struct {
		name        string
		input       io.Reader
		stderr      io.Writer
		want        bool
		wantErr     error
		wantPrompts int
	}{
		{name: "accepted", input: strings.NewReader("y\n"), stderr: &bytes.Buffer{}, want: true, wantPrompts: 1},
		{name: "declined", input: strings.NewReader("n\n"), stderr: &bytes.Buffer{}, wantPrompts: 1},
		{name: "invalid then accepted", input: strings.NewReader("perhaps\ny\n"), stderr: &bytes.Buffer{}, want: true, wantPrompts: 2},
		{name: "EOF", input: strings.NewReader(""), stderr: &bytes.Buffer{}, wantPrompts: 1},
		{name: "read failure", input: readError{err: readErr}, stderr: &bytes.Buffer{}, wantErr: readErr, wantPrompts: 1},
		{name: "write failure", input: strings.NewReader("y\n"), stderr: writeError{err: writeErr}, wantErr: writeErr},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ios := NewIOStreams(tc.input, io.Discard, tc.stderr)
			got, err := ios.ConfirmCreate("config.yaml")
			if got != tc.want {
				t.Errorf("ConfirmCreate() = %v, want %v", got, tc.want)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("ConfirmCreate() error = %v, want %v", err, tc.wantErr)
			}
			if buf, ok := tc.stderr.(*bytes.Buffer); ok {
				const prompt = "config.yaml does not exist. Create it? [yn]:"
				if got := strings.Count(buf.String(), prompt); got != tc.wantPrompts {
					t.Errorf("prompt count = %d, want %d; output %q", got, tc.wantPrompts, buf.String())
				}
			}
		})
	}
}

func TestRuntimeAskToCreate(t *testing.T) {
	t.Parallel()

	permissionErr := errors.New("permission denied")
	tests := []struct {
		name    string
		path    string
		statErr error
		input   string
		want    bool
		wantErr error
	}{
		{name: "empty path", wantErr: errors.New("path cannot be empty")},
		{name: "existing file", path: "config.yaml", wantErr: ErrFileExists},
		{name: "missing accepted", path: "config.yaml", statErr: os.ErrNotExist, input: "y\n", want: true},
		{name: "missing declined", path: "config.yaml", statErr: os.ErrNotExist, input: "n\n"},
		{name: "stat failure", path: "config.yaml", statErr: permissionErr, wantErr: permissionErr},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			statCalls := 0
			rt := NewTestRuntime(strings.NewReader(tc.input), io.Discard, io.Discard)
			rt.FileCreation = fileCreationOperationsStub{
				stat: func(string) error {
					statCalls++
					return tc.statErr
				},
			}

			got, err := rt.AskToCreate(tc.path)
			if got != tc.want {
				t.Errorf("AskToCreate() = %v, want %v", got, tc.want)
			}
			if tc.name == "empty path" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr.Error()) {
					t.Errorf("AskToCreate() error = %v, want message containing %q", err, tc.wantErr)
				}
				if statCalls != 0 {
					t.Errorf("Stat calls = %d, want 0", statCalls)
				}
			} else if !errors.Is(err, tc.wantErr) {
				t.Errorf("AskToCreate() error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestRuntimeCreateIfNotExists_Failures(t *testing.T) {
	t.Parallel()

	statErr := errors.New("stat failed")
	mkdirErr := errors.New("mkdir failed")
	openErr := errors.New("open failed")
	closeErr := errors.New("close failed")
	tests := []struct {
		name    string
		path    string
		ops     fileCreationOperationsStub
		wantErr error
	}{
		{name: "empty path", wantErr: errors.New("path cannot be empty")},
		{
			name: "existing file",
			path: "config.yaml",
			ops:  fileCreationOperationsStub{stat: func(string) error { return nil }},
		},
		{
			name:    "stat failure",
			path:    "config.yaml",
			ops:     fileCreationOperationsStub{stat: func(string) error { return statErr }},
			wantErr: statErr,
		},
		{
			name: "parent creation failure",
			path: "parent/config.yaml",
			ops: fileCreationOperationsStub{
				stat:     func(string) error { return os.ErrNotExist },
				mkdirAll: func(string, os.FileMode) error { return mkdirErr },
			},
			wantErr: mkdirErr,
		},
		{
			name: "file open failure",
			path: "parent/config.yaml",
			ops: fileCreationOperationsStub{
				stat:     func(string) error { return os.ErrNotExist },
				mkdirAll: func(string, os.FileMode) error { return nil },
				openFile: func(string, int, os.FileMode) (io.Closer, error) { return nil, openErr },
			},
			wantErr: openErr,
		},
		{
			name: "file close failure",
			path: "parent/config.yaml",
			ops: fileCreationOperationsStub{
				stat:     func(string) error { return os.ErrNotExist },
				mkdirAll: func(string, os.FileMode) error { return nil },
				openFile: func(string, int, os.FileMode) (io.Closer, error) {
					return closeFunc(func() error { return closeErr }), nil
				},
			},
			wantErr: closeErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rt := NewTestRuntime(nil, io.Discard, io.Discard)
			if tc.path != "" {
				rt.FileCreation = tc.ops
			}
			err := rt.CreateIfNotExists(tc.path)
			if tc.path == "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr.Error()) {
					t.Errorf("CreateIfNotExists() error = %v, want message containing %q", err, tc.wantErr)
				}
			} else if !errors.Is(err, tc.wantErr) {
				t.Errorf("CreateIfNotExists() error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestRuntimeCreateIfNotExists_Operations(t *testing.T) {
	t.Parallel()

	var operations []string
	rt := NewTestRuntime(nil, io.Discard, io.Discard)
	rt.FileCreation = fileCreationOperationsStub{
		stat: func(path string) error {
			operations = append(operations, "stat:"+path)
			return os.ErrNotExist
		},
		mkdirAll: func(path string, mode os.FileMode) error {
			operations = append(operations, "mkdir:"+path)
			if mode != 0o755 {
				t.Errorf("MkdirAll mode = %o, want 755", mode)
			}
			return nil
		},
		openFile: func(path string, flag int, mode os.FileMode) (io.Closer, error) {
			operations = append(operations, "open:"+path)
			if flag != os.O_RDONLY|os.O_CREATE {
				t.Errorf("OpenFile flag = %d, want %d", flag, os.O_RDONLY|os.O_CREATE)
			}
			if mode != 0o644 {
				t.Errorf("OpenFile mode = %o, want 644", mode)
			}
			return closeFunc(func() error {
				operations = append(operations, "close")
				return nil
			}), nil
		},
	}

	if err := rt.CreateIfNotExists("parent/config.yaml"); err != nil {
		t.Fatalf("CreateIfNotExists() error = %v", err)
	}
	want := []string{
		"stat:parent/config.yaml",
		"mkdir:parent",
		"open:parent/config.yaml",
		"close",
	}
	if strings.Join(operations, ",") != strings.Join(want, ",") {
		t.Errorf("operations = %v, want %v", operations, want)
	}
}

func TestRuntimeCreateIfNotExists_NestedFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "parent", "nested", "config.yaml")
	rt := NewTestRuntime(nil, io.Discard, io.Discard)
	if err := rt.CreateIfNotExists(path); err != nil {
		t.Fatalf("CreateIfNotExists() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(created file) error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Errorf("created file mode = %o, want 644", got)
	}
	if err := rt.CreateIfNotExists(path); err != nil {
		t.Errorf("CreateIfNotExists(existing file) error = %v", err)
	}
}
