// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type recordingTemporaryFile struct {
	operations *[]string
	path       string
	mode       os.FileMode
	chmodErr   error
	writeErr   error
	shortWrite bool
	syncErr    error
	closeErr   error
}

func (f *recordingTemporaryFile) record(operation string) {
	*f.operations = append(*f.operations, operation)
}

func (f *recordingTemporaryFile) Name() string {
	f.record("name")
	return f.path
}

func (f *recordingTemporaryFile) Chmod(mode os.FileMode) error {
	f.record("chmod")
	f.mode = mode
	return f.chmodErr
}

func (f *recordingTemporaryFile) Write(data []byte) (int, error) {
	f.record("write")
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	if f.shortWrite {
		return len(data) - 1, nil
	}
	return len(data), nil
}

func (f *recordingTemporaryFile) Sync() error {
	f.record("sync")
	return f.syncErr
}

func (f *recordingTemporaryFile) Close() error {
	f.record("close")
	return f.closeErr
}

// TestWriteFileDurability_Order verifies that writeFile writes, syncs, and
// closes the temporary file before renaming it into place and syncing the
// directory, and gives it the existing file's mode.
func TestWriteFileDurability_Order(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("old"), 0o640); err != nil {
		t.Fatal(err)
	}

	var operations []string
	file := &recordingTemporaryFile{
		operations: &operations,
		path:       filepath.Join(dir, ".config.yaml.test"),
	}
	err := writeFile(path, []byte("default-cluster: demo\n"), fileWriteOps{
		stat: func(name string) (os.FileInfo, error) {
			operations = append(operations, "stat")
			return os.Stat(name)
		},
		createTemp: func(string, string) (temporaryFile, error) {
			operations = append(operations, "create")
			return file, nil
		},
		rename: func(string, string) error {
			operations = append(operations, "rename")
			return nil
		},
		remove: func(string) error {
			operations = append(operations, "remove")
			return nil
		},
		syncDir: func(string) error {
			operations = append(operations, "sync-dir")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("writeFile() error = %v", err)
	}

	want := []string{"stat", "create", "name", "chmod", "write", "sync", "close", "rename", "sync-dir", "remove"}
	if !reflect.DeepEqual(operations, want) {
		t.Errorf("operations = %v, want %v", operations, want)
	}
	if file.mode.Perm() != 0o640 {
		t.Errorf("temporary mode = %o, want preserved mode 0640", file.mode.Perm())
	}
}

// TestWriteFile_SyncsParentDirectoryAfterRename verifies that writeFile syncs
// the config file's parent directory.
func TestWriteFile_SyncsParentDirectoryAfterRename(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	var synced string
	err := writeFile(path, []byte("new"), fileWriteOps{
		stat: os.Stat, createTemp: createTemporaryFile, rename: os.Rename,
		remove:  os.Remove,
		syncDir: func(path string) error { synced = path; return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if synced != filepath.Dir(path) {
		t.Fatalf("synced directory = %q, want %q", synced, filepath.Dir(path))
	}
}
