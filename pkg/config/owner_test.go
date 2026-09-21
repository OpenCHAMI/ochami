// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build unix

package config

import (
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"time"
)

// ownedFileInfo is an os.FileInfo that reports a chosen mode and owner.
type ownedFileInfo struct {
	mode     os.FileMode
	uid, gid uint32
}

func (fi ownedFileInfo) Name() string       { return "config.yaml" }
func (fi ownedFileInfo) Size() int64        { return 0 }
func (fi ownedFileInfo) Mode() os.FileMode  { return fi.mode }
func (fi ownedFileInfo) ModTime() time.Time { return time.Time{} }
func (fi ownedFileInfo) IsDir() bool        { return false }
func (fi ownedFileInfo) Sys() any           { return &syscall.Stat_t{Uid: fi.uid, Gid: fi.gid} }

// ownerTestOps returns write primitives that record each operation, report
// existing for the destination, and hand out file as the temporary file.
func ownerTestOps(operations *[]string, existing os.FileInfo, file *recordingTemporaryFile) fileWriteOps {
	return fileWriteOps{
		stat: func(string) (os.FileInfo, error) {
			*operations = append(*operations, "stat")
			return existing, nil
		},
		createTemp: func(string, string, os.FileMode) (temporaryFile, error) {
			*operations = append(*operations, "create")
			return file, nil
		},
		rename:  func(string, string) error { *operations = append(*operations, "rename"); return nil },
		remove:  func(string) error { *operations = append(*operations, "remove"); return nil },
		syncDir: func(string) error { *operations = append(*operations, "sync-dir"); return nil },
	}
}

// TestFileOwner verifies that fileOwner reports the user and group IDs that
// own a real file.
func TestFileOwner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	st := info.Sys().(*syscall.Stat_t)

	uid, gid, ok := fileOwner(info)
	if !ok {
		t.Fatal("fileOwner() ok = false, want true")
	}
	if uid != os.Getuid() || uid != int(st.Uid) || gid != int(st.Gid) {
		t.Errorf("fileOwner() = %d:%d, want %d:%d", uid, gid, st.Uid, st.Gid)
	}
}

// TestWriteFile_PreservesOwner verifies that writeFile gives the temporary
// file the existing config file's owner and group, before its mode, when
// they differ from the temporary file's.
func TestWriteFile_PreservesOwner(t *testing.T) {
	existing := ownedFileInfo{mode: 0o640, uid: 1234, gid: 5678}
	var operations []string
	file := &recordingTemporaryFile{
		operations: &operations,
		path:       filepath.Join(t.TempDir(), ".config.yaml.test"),
		info:       ownedFileInfo{mode: 0o600, uid: 1, gid: 2},
	}

	if err := writeFile("config.yaml", []byte("new"), ownerTestOps(&operations, existing, file)); err != nil {
		t.Fatalf("writeFile() error = %v", err)
	}

	want := []string{"stat", "create", "name", "fstat", "chown", "chmod", "write", "sync", "close", "rename", "sync-dir", "remove"}
	if !reflect.DeepEqual(operations, want) {
		t.Errorf("operations = %v, want %v", operations, want)
	}
	if !file.chowned || file.uid != 1234 || file.gid != 5678 {
		t.Errorf("temporary owner = %d:%d (chowned %v), want 1234:5678", file.uid, file.gid, file.chowned)
	}
	if file.mode.Perm() != 0o640 {
		t.Errorf("temporary mode = %o, want preserved mode 0640", file.mode.Perm())
	}
}

// TestWriteFile_OnlyGroupDiffers verifies that writeFile restores the
// existing config file's group when the temporary file has the right owner
// but a different group (e.g. one inherited from a set-group-ID directory).
func TestWriteFile_OnlyGroupDiffers(t *testing.T) {
	existing := ownedFileInfo{mode: 0o640, uid: 1234, gid: 5678}
	var operations []string
	file := &recordingTemporaryFile{
		operations: &operations,
		path:       filepath.Join(t.TempDir(), ".config.yaml.test"),
		info:       ownedFileInfo{mode: 0o600, uid: 1234, gid: 9},
	}

	if err := writeFile("config.yaml", []byte("new"), ownerTestOps(&operations, existing, file)); err != nil {
		t.Fatalf("writeFile() error = %v", err)
	}
	if !file.chowned || file.uid != 1234 || file.gid != 5678 {
		t.Errorf("temporary owner = %d:%d (chowned %v), want 1234:5678", file.uid, file.gid, file.chowned)
	}
}
