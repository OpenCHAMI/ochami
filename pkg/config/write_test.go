// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type recordingTemporaryFile struct {
	operations *[]string
	path       string
	mode       os.FileMode
	info       os.FileInfo // returned by Stat
	statErr    error
	uid, gid   int
	chowned    bool
	chownErr   error
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

func (f *recordingTemporaryFile) Stat() (os.FileInfo, error) {
	f.record("fstat")
	return f.info, f.statErr
}

func (f *recordingTemporaryFile) Chown(uid, gid int) error {
	f.record("chown")
	f.uid, f.gid, f.chowned = uid, gid, true
	return f.chownErr
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

// withOwnerCheck returns ops with the "fstat" that preserveOwner records
// inserted before "chmod", if the platform reports existing's owner.
func withOwnerCheck(ops []string, existing os.FileInfo) []string {
	if _, _, ok := fileOwner(existing); !ok {
		return ops
	}
	var out []string
	for _, op := range ops {
		if op == "chmod" {
			out = append(out, "fstat")
		}
		out = append(out, op)
	}
	return out
}

// TestWriteFileDurability_Order verifies that writeFile writes, syncs, and
// closes the temporary file before renaming it into place and syncing the
// directory, creates it private, and gives it the existing file's mode
// without changing an owner that already matches.
func TestWriteFileDurability_Order(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("old"), 0o640); err != nil {
		t.Fatal(err)
	}
	existing, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	var operations []string
	file := &recordingTemporaryFile{
		operations: &operations,
		path:       filepath.Join(dir, ".config.yaml.test"),
		info:       existing,
	}
	var createPerm os.FileMode
	err = writeFile(path, []byte("default-cluster: demo\n"), fileWriteOps{
		stat: func(name string) (os.FileInfo, error) {
			operations = append(operations, "stat")
			return os.Stat(name)
		},
		createTemp: func(_, _ string, perm os.FileMode) (temporaryFile, error) {
			operations = append(operations, "create")
			createPerm = perm
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

	want := withOwnerCheck([]string{"stat", "create", "name", "chmod", "write", "sync", "close", "rename", "sync-dir", "remove"}, existing)
	if !reflect.DeepEqual(operations, want) {
		t.Errorf("operations = %v, want %v", operations, want)
	}
	if createPerm != 0o600 {
		t.Errorf("temporary file created with mode %o, want 0600", createPerm)
	}
	if file.mode.Perm() != 0o640 {
		t.Errorf("temporary mode = %o, want preserved mode 0640", file.mode.Perm())
	}
}

// TestWriteFileDurability_NewFile verifies that writeFile creates the
// temporary file for a new config file with mode 0644, for the umask to
// reduce, and neither changes its owner nor sets its mode afterwards.
func TestWriteFileDurability_NewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	var operations []string
	file := &recordingTemporaryFile{
		operations: &operations,
		path:       filepath.Join(dir, ".config.yaml.test"),
	}
	var createPerm os.FileMode
	err := writeFile(path, []byte("default-cluster: demo\n"), fileWriteOps{
		stat: func(name string) (os.FileInfo, error) {
			operations = append(operations, "stat")
			return os.Stat(name)
		},
		createTemp: func(_, _ string, perm os.FileMode) (temporaryFile, error) {
			operations = append(operations, "create")
			createPerm = perm
			return file, nil
		},
		rename:  func(string, string) error { operations = append(operations, "rename"); return nil },
		remove:  func(string) error { operations = append(operations, "remove"); return nil },
		syncDir: func(string) error { operations = append(operations, "sync-dir"); return nil },
	})
	if err != nil {
		t.Fatalf("writeFile() error = %v", err)
	}

	want := []string{"stat", "create", "name", "write", "sync", "close", "rename", "sync-dir", "remove"}
	if !reflect.DeepEqual(operations, want) {
		t.Errorf("operations = %v, want %v", operations, want)
	}
	if createPerm != 0o644 {
		t.Errorf("temporary file created with mode %o, want 0644", createPerm)
	}
}

// TestWriteFile_NewFileModeFollowsUmask verifies that a newly written config
// file gets the same mode as a file created with mode 0644 in the same
// directory, i.e. 0644 reduced by the process umask.
func TestWriteFile_NewFileModeFollowsUmask(t *testing.T) {
	dir := t.TempDir()
	ref := filepath.Join(dir, "reference")
	rf, err := os.OpenFile(ref, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if err := rf.Close(); err != nil {
		t.Fatal(err)
	}
	refInfo, err := os.Stat(ref)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "config.yaml")
	if err := writeFile(path, []byte("new"), defaultWriteOps); err != nil {
		t.Fatalf("writeFile() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != refInfo.Mode().Perm() {
		t.Errorf("new config mode = %o, want %o (0644 reduced by the umask)", info.Mode().Perm(), refInfo.Mode().Perm())
	}
}

// TestCreateTemporaryFile_NameAndMode verifies that createTemporaryFile
// creates a new file in dir whose name replaces the pattern's last "*", with
// the same mode as a file created with perm in that directory.
func TestCreateTemporaryFile_NameAndMode(t *testing.T) {
	dir := t.TempDir()
	tmp, err := createTemporaryFile(dir, ".a*b.yaml.*", 0o640)
	if err != nil {
		t.Fatalf("createTemporaryFile() error = %v", err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatal(err)
	}
	name := tmp.Name()
	if filepath.Dir(name) != dir {
		t.Errorf("temporary file %q not in %q", name, dir)
	}
	base := filepath.Base(name)
	if !strings.HasPrefix(base, ".a*b.yaml.") || len(base) == len(".a*b.yaml.") {
		t.Errorf("temporary file name = %q, want .a*b.yaml.<random>", base)
	}

	ref := filepath.Join(dir, "reference")
	rf, err := os.OpenFile(ref, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		t.Fatal(err)
	}
	if err := rf.Close(); err != nil {
		t.Fatal(err)
	}
	refInfo, err := os.Stat(ref)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(name)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != refInfo.Mode().Perm() {
		t.Errorf("temporary file mode = %o, want %o", info.Mode().Perm(), refInfo.Mode().Perm())
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
