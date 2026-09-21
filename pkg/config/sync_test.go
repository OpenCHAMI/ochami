// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build !windows

package config

// sync_test.go contains tests for sync-related functions.

import (
	"testing"
)

// TestSyncParentDirectory_Success verifies that syncParentDirectory succeeds
// for an existing directory.
func TestSyncParentDirectory_Success(t *testing.T) {
	// Create a temporary directory
	dir := t.TempDir()

	// syncParentDirectory should succeed on the temp directory
	err := syncParentDirectory(dir)
	if err != nil {
		t.Fatalf("syncParentDirectory failed on valid directory: %v", err)
	}
}
