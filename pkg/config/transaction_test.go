// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"path/filepath"
	"testing"
)

// unwritablePath returns a path whose parent is a regular file, so writing
// to it fails deterministically regardless of the test's user or umask
// (unlike a read-only directory, which root bypasses).
func unwritablePath(t *testing.T) string {
	t.Helper()
	notADir := filepath.Join(t.TempDir(), "not-a-dir")
	if _, err := CreateFile(notADir); err != nil {
		t.Fatalf("CreateFile(%q): %v", notADir, err)
	}
	return filepath.Join(notADir, "config.yaml")
}

// TestFile_UpdatePersistsAllChanges verifies that a successful Update writes
// all of its batched mutations to disk.
func TestFile_UpdatePersistsAllChanges(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	f, err := CreateFile(path)
	if err != nil {
		t.Fatalf("CreateFile(): %v", err)
	}

	if err := f.Update(func(f *File) error {
		if err := f.SetClusterKey("c1", "cluster.uri", "https://c1"); err != nil {
			return err
		}
		return f.SetDefaultCluster("c1")
	}); err != nil {
		t.Fatalf("Update(): %v", err)
	}

	reopened, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile(): %v", err)
	}
	if _, err := reopened.Cluster("c1"); err != nil {
		t.Fatalf("cluster \"c1\" not persisted by successful Update: %v", err)
	}
	if got := reopened.Get("default-cluster"); got != "c1" {
		t.Fatalf("persisted default-cluster = %v, want %q", got, "c1")
	}
}

// TestFile_UpdateReadsSeeStagedChanges verifies that Get, Raw, Cluster, and
// Clusters called inside Update observe the changes fn has made so far, before
// they are written.
func TestFile_UpdateReadsSeeStagedChanges(t *testing.T) {
	t.Parallel()

	f, err := CreateFile(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("CreateFile(): %v", err)
	}

	if err := f.Update(func(f *File) error {
		if err := f.SetClusterKey("c1", "cluster.uri", "https://c1"); err != nil {
			return err
		}
		if err := f.SetKey("log.level", "debug"); err != nil {
			return err
		}
		if _, err := f.Cluster("c1"); err != nil {
			t.Errorf("Cluster(%q) inside Update = %v, want the staged cluster", "c1", err)
		}
		if clusters, err := f.Clusters(); err != nil || len(clusters) != 1 {
			t.Errorf("Clusters() inside Update = (%v, %v), want the one staged cluster", clusters, err)
		}
		if got := f.Get("log.level"); got != "debug" {
			t.Errorf("Get(%q) inside Update = %v, want %q", "log.level", got, "debug")
		}
		if _, ok := f.Raw()["log"]; !ok {
			t.Errorf("Raw() inside Update = %v, want the staged log key", f.Raw())
		}
		return nil
	}); err != nil {
		t.Fatalf("Update(): %v", err)
	}
}
