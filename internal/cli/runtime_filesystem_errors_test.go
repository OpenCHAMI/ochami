// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

// TestRuntimeCreateIfNotExists_Failures verifies that CreateIfNotExists rejects
// an empty path, leaves an existing file alone, and returns a failure to check
// the file, create its parent directories, or create or close the file.
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
