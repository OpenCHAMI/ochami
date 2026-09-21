// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build !windows

package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSyncParentDirectory_NonExistent verifies that syncParentDirectory fails
// on a non-existent directory.
func TestSyncParentDirectory_NonExistent(t *testing.T) {
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
