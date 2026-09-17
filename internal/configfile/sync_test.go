// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build !windows

package configfile

// sync_test.go contains tests for sync-related functions.

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSyncParentDirectorySuccess verifies that syncParentDirectory works correctly
// on a valid directory.
func TestSyncParentDirectorySuccess(t *testing.T) {
	// Create a temporary directory
	dir := t.TempDir()

	// syncParentDirectory should succeed on the temp directory
	err := syncParentDirectory(dir)
	if err != nil {
		t.Fatalf("syncParentDirectory failed on valid directory: %v", err)
	}
}

// TestSyncParentDirectoryNonExistent verifies that syncParentDirectory fails
// on a non-existent directory.
func TestSyncParentDirectoryNonExistent(t *testing.T) {
	// Try to sync a non-existent directory
	nonExistent := filepath.Join(t.TempDir(), "nonexistent")

	err := syncParentDirectory(nonExistent)
	if err == nil {
		t.Error("expected error for non-existent directory, got nil")
	}

	// Should get a path error
	if !os.IsNotExist(err) {
		t.Errorf("expected os.ErrNotExist, got: %v", err)
	}
}
