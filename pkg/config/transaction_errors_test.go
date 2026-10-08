// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"path/filepath"
	"testing"
)

// TestFile_FailedWriteLeavesStateUnchanged verifies that when a mutating
// method's write fails, neither the file on disk nor the File's in-memory
// state reflects the attempted change: the mutation is staged on a clone that
// is discarded on failure.
func TestFile_FailedWriteLeavesStateUnchanged(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	f, err := CreateFile(path)
	if err != nil {
		t.Fatalf("CreateFile(): %v", err)
	}
	if err := f.SetKey("default-cluster", "good"); err != nil {
		t.Fatalf("SetKey(): %v", err)
	}

	// Redirect writes to a path that can never be created, so the next save
	// fails after the mutation has been staged.
	f.path = unwritablePath(t)
	if err := f.SetKey("default-cluster", "bad"); err == nil {
		t.Fatal("SetKey(): expected a write error, got nil")
	}

	// The failed mutation must not survive in memory.
	if got := f.Get("default-cluster"); got != "good" {
		t.Fatalf("in-memory default-cluster = %v, want %q (failed write must not change state)", got, "good")
	}

	// A subsequent successful write must persist only the new change, not the
	// discarded one. Point back at a writable path and commit a fresh value.
	f.path = path
	if err := f.SetKey("log.level", "debug"); err != nil {
		t.Fatalf("SetKey() after recovery: %v", err)
	}

	reopened, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile(): %v", err)
	}
	if got := reopened.Get("default-cluster"); got != "good" {
		t.Fatalf("persisted default-cluster = %v, want %q (discarded write must not reach disk)", got, "good")
	}
	if got := reopened.Get("log.level"); got != "debug" {
		t.Fatalf("persisted log.level = %v, want %q", got, "debug")
	}
}

// TestFile_FailedUpdateLeavesStateUnchanged verifies that a batched Update
// whose final write fails leaves the File's in-memory state exactly as it was
// before Update, so none of the batch's staged mutations survive.
func TestFile_FailedUpdateLeavesStateUnchanged(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	f, err := CreateFile(path)
	if err != nil {
		t.Fatalf("CreateFile(): %v", err)
	}
	if err := f.AddCluster("foo", ClusterConfig{URI: "https://foo"}); err != nil {
		t.Fatalf("AddCluster(): %v", err)
	}

	f.path = unwritablePath(t)
	err = f.Update(func(f *File) error {
		if err := f.SetClusterKey("bar", "cluster.uri", "https://bar"); err != nil {
			return err
		}
		return f.SetDefaultCluster("bar")
	})
	if err == nil {
		t.Fatal("Update(): expected a write error, got nil")
	}

	// None of the batched mutations may survive in memory.
	if _, err := f.Cluster("bar"); err == nil {
		t.Fatal("cluster \"bar\" present after failed Update; batch must be discarded")
	}
	if got := f.Get("default-cluster"); got != nil {
		t.Fatalf("default-cluster = %v, want unset after failed Update", got)
	}
	if _, err := f.Cluster("foo"); err != nil {
		t.Fatalf("pre-existing cluster \"foo\" lost after failed Update: %v", err)
	}
}

// TestFile_RejectedMutationLeavesNoStagedState verifies that a mutation that
// rejects its input leaves no working copy behind, so a later mutation starts
// from the committed state.
func TestFile_RejectedMutationLeavesNoStagedState(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	f, err := CreateFile(path)
	if err != nil {
		t.Fatalf("CreateFile(): %v", err)
	}

	if err := f.SetDefaultCluster("missing"); err == nil {
		t.Fatal("SetDefaultCluster(): expected ErrUnknownCluster, got nil")
	}
	if err := f.SetKey("clusters", "x"); err == nil {
		t.Fatal("SetKey(\"clusters\"): expected rejection, got nil")
	}
	if f.staging != nil {
		t.Fatal("staging is non-nil after rejected mutations; want it discarded")
	}

	if err := f.SetKey("log.level", "debug"); err != nil {
		t.Fatalf("SetKey(): %v", err)
	}
	reopened, err := OpenFile(path)
	if err != nil {
		t.Fatalf("OpenFile(): %v", err)
	}
	if got := reopened.Raw(); len(got) != 1 || reopened.Get("log.level") != "debug" {
		t.Fatalf("persisted config = %v, want only log.level", got)
	}
}
