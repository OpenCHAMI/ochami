// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestWriteFileDurability_Failures verifies that writeFile returns the failure
// of each step (creating, chmodding, writing, syncing, or closing the temporary
// file, renaming it, or syncing the directory) and the cleanup it performs
// after each one.
func TestWriteFileDurability_Failures(t *testing.T) {
	chmodErr := errors.New("chmod failed")
	writeErr := errors.New("write failed")
	syncErr := errors.New("sync failed")
	closeErr := errors.New("close failed")
	createErr := errors.New("create failed")
	renameErr := errors.New("rename failed")
	directorySyncErr := errors.New("directory sync failed")
	cleanupErr := errors.New("cleanup failed")

	tests := []struct {
		name       string
		createErr  error
		configure  func(*recordingTemporaryFile)
		renameErr  error
		syncDirErr error
		wantErr    error
		wantOps    []string
	}{
		{
			name:      "create temporary file",
			createErr: createErr, wantErr: createErr,
			wantOps: []string{"stat", "create"},
		},
		{
			name:      "chmod",
			configure: func(file *recordingTemporaryFile) { file.chmodErr = chmodErr },
			wantErr:   chmodErr,
			wantOps:   []string{"stat", "create", "name", "chmod", "close", "remove"},
		},
		{
			name:      "short write",
			configure: func(file *recordingTemporaryFile) { file.shortWrite = true },
			wantErr:   io.ErrShortWrite,
			wantOps:   []string{"stat", "create", "name", "chmod", "write", "close", "remove"},
		},
		{
			name:      "write",
			configure: func(file *recordingTemporaryFile) { file.writeErr = writeErr },
			wantErr:   writeErr,
			wantOps:   []string{"stat", "create", "name", "chmod", "write", "close", "remove"},
		},
		{
			name:      "file sync",
			configure: func(file *recordingTemporaryFile) { file.syncErr = syncErr },
			wantErr:   syncErr,
			wantOps:   []string{"stat", "create", "name", "chmod", "write", "sync", "close", "remove"},
		},
		{
			name:      "close",
			configure: func(file *recordingTemporaryFile) { file.closeErr = closeErr },
			wantErr:   closeErr,
			wantOps:   []string{"stat", "create", "name", "chmod", "write", "sync", "close", "remove"},
		},
		{
			name: "rename", renameErr: renameErr, wantErr: renameErr,
			wantOps: []string{"stat", "create", "name", "chmod", "write", "sync", "close", "rename", "remove"},
		},
		{
			name: "parent directory sync", syncDirErr: directorySyncErr, wantErr: directorySyncErr,
			wantOps: []string{"stat", "create", "name", "chmod", "write", "sync", "close", "rename", "sync-dir", "remove"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
				t.Fatal(err)
			}

			var operations []string
			file := &recordingTemporaryFile{
				operations: &operations,
				path:       filepath.Join(dir, ".config.yaml.test"),
			}
			if tt.configure != nil {
				tt.configure(file)
			}
			ops := fileWriteOps{
				stat: func(name string) (os.FileInfo, error) {
					operations = append(operations, "stat")
					return os.Stat(name)
				},
				createTemp: func(gotDir, pattern string) (temporaryFile, error) {
					operations = append(operations, "create")
					if gotDir != dir {
						t.Errorf("temporary directory = %q, want %q", gotDir, dir)
					}
					if pattern != ".config.yaml.*" {
						t.Errorf("temporary pattern = %q, want .config.yaml.*", pattern)
					}
					if tt.createErr != nil {
						return nil, tt.createErr
					}
					return file, nil
				},
				rename: func(oldPath, newPath string) error {
					operations = append(operations, "rename")
					if oldPath != file.path || newPath != path {
						t.Errorf("rename(%q, %q), want (%q, %q)", oldPath, newPath, file.path, path)
					}
					return tt.renameErr
				},
				remove: func(name string) error {
					operations = append(operations, "remove")
					if name != file.path {
						t.Errorf("remove(%q), want %q", name, file.path)
					}
					return cleanupErr
				},
				syncDir: func(name string) error {
					operations = append(operations, "sync-dir")
					if name != dir {
						t.Errorf("syncDir(%q), want %q", name, dir)
					}
					return tt.syncDirErr
				},
			}

			err := writeFile(path, []byte("default-cluster: demo\n"), ops)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("writeFile() error = %v, want error wrapping %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(operations, tt.wantOps) {
				t.Errorf("operations = %v, want %v", operations, tt.wantOps)
			}
			if tt.createErr == nil && file.mode.Perm() != 0o600 {
				t.Errorf("temporary mode = %o, want preserved mode 0600", file.mode.Perm())
			}
			if errors.Is(err, cleanupErr) {
				t.Errorf("cleanup error replaced operation error: %v", err)
			}
		})
	}
}

// TestWriteFile_FailedTemporaryWritePreservesOldFile verifies that a failure to
// create the temporary file leaves the existing config file unchanged.
func TestWriteFile_FailedTemporaryWritePreservesOldFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	opErr := errors.New("temporary write failed")
	err := writeFile(path, []byte("new"), fileWriteOps{
		stat: os.Stat,
		createTemp: func(string, string) (temporaryFile, error) {
			return nil, opErr
		},
		rename: os.Rename, remove: os.Remove, syncDir: syncParentDirectory,
	})
	if !errors.Is(err, opErr) {
		t.Fatalf("writeFile() error = %v, want injected error", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "old" {
		t.Fatalf("old config changed after temporary-write failure: %q, %v", got, err)
	}
}

// TestWriteFile_RenameFailureCleansTemporaryFile verifies that a failed rename
// leaves the existing config file unchanged and removes the temporary file.
func TestWriteFile_RenameFailureCleansTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	renameErr := errors.New("rename failed")
	err := writeFile(path, []byte("new"), fileWriteOps{
		stat: os.Stat, createTemp: createTemporaryFile,
		rename: func(string, string) error { return renameErr },
		remove: os.Remove, syncDir: syncParentDirectory,
	})
	if !errors.Is(err, renameErr) {
		t.Fatalf("writeFile() error = %v, want rename error", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "old" {
		t.Fatalf("old config changed after rename failure: %q, %v", got, err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".config.yaml.*"))
	if err != nil || len(matches) != 0 {
		t.Fatalf("temporary files after failure = %v, %v", matches, err)
	}
}

// TestWriteFile_DirectorySyncFailureIsReturned verifies that writeFile returns
// a failure to sync the parent directory after the rename.
func TestWriteFile_DirectorySyncFailureIsReturned(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	syncErr := errors.New("directory sync failed")
	err := writeFile(path, []byte("new"), fileWriteOps{
		stat: os.Stat, createTemp: createTemporaryFile, rename: os.Rename,
		remove: os.Remove, syncDir: func(string) error { return syncErr },
	})
	if !errors.Is(err, syncErr) {
		t.Fatalf("writeFile() error = %v, want sync error", err)
	}
}
