// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package configfile

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"

	"github.com/openchami/ochami/pkg/config"
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

func writeTestConfig(t *testing.T) *koanf.Koanf {
	t.Helper()
	ko := koanf.NewWithConf(kConfig)
	if err := ko.Load(confmap.Provider(config.DefaultGlobalMap(), "."), nil); err != nil {
		t.Fatalf("load test config: %v", err)
	}
	return ko
}

func TestWriteConfigDurability_Failures(t *testing.T) {
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
			ops := configWriteOps{
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

			err := writeConfig(path, writeTestConfig(t), ops)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("writeConfig() error = %v, want error wrapping %v", err, tt.wantErr)
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

func TestWriteConfigDurability_Order(t *testing.T) {
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
	err := writeConfig(path, writeTestConfig(t), configWriteOps{
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
		t.Fatalf("writeConfig() error = %v", err)
	}

	want := []string{"stat", "create", "name", "chmod", "write", "sync", "close", "rename", "sync-dir", "remove"}
	if !reflect.DeepEqual(operations, want) {
		t.Errorf("operations = %v, want %v", operations, want)
	}
	if file.mode.Perm() != 0o640 {
		t.Errorf("temporary mode = %o, want preserved mode 0640", file.mode.Perm())
	}
}
