// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build windows

package config

import "testing"

// TestSyncParentDirectory_IsNoOp verifies that on Windows syncParentDirectory
// does nothing and succeeds, even for a directory that doesn't exist.
func TestSyncParentDirectory_IsNoOp(t *testing.T) {
	t.Parallel()

	if err := syncParentDirectory(`Z:\path\that\need\not\exist`); err != nil {
		t.Fatalf("syncParentDirectory() error = %v, want nil", err)
	}
}
