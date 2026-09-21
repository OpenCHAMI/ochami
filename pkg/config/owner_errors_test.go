// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build unix

package config

import (
	"errors"
	"io/fs"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
)

// TestWriteFile_OwnerFailures verifies that writeFile refuses to replace the
// config file when it can't give the replacement the existing file's owner,
// whether checking the temporary file's owner or changing it fails, and
// removes the temporary file without renaming it into place.
func TestWriteFile_OwnerFailures(t *testing.T) {
	statErr := errors.New("fstat failed")
	tests := []struct {
		name      string
		configure func(*recordingTemporaryFile)
		wantErr   error
		wantOps   []string
	}{
		{
			name:      "check owner",
			configure: func(file *recordingTemporaryFile) { file.statErr = statErr },
			wantErr:   statErr,
			wantOps:   []string{"stat", "create", "name", "fstat", "close", "remove"},
		},
		{
			name:      "change owner not permitted",
			configure: func(file *recordingTemporaryFile) { file.chownErr = syscall.EPERM },
			wantErr:   fs.ErrPermission,
			wantOps:   []string{"stat", "create", "name", "fstat", "chown", "close", "remove"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existing := ownedFileInfo{mode: 0o640, uid: 1234, gid: 5678}
			var operations []string
			file := &recordingTemporaryFile{
				operations: &operations,
				path:       filepath.Join(t.TempDir(), ".config.yaml.test"),
				info:       ownedFileInfo{mode: 0o600, uid: 1, gid: 2},
			}
			tt.configure(file)

			err := writeFile("config.yaml", []byte("new"), ownerTestOps(&operations, existing, file))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("writeFile() error = %v, want error wrapping %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(operations, tt.wantOps) {
				t.Errorf("operations = %v, want %v", operations, tt.wantOps)
			}
		})
	}
}
