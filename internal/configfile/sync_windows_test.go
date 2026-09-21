// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build windows

package configfile

import "testing"

func TestSyncParentDirectoryIsNoOp(t *testing.T) {
	t.Parallel()

	if err := syncParentDirectory(`Z:\path\that\need\not\exist`); err != nil {
		t.Fatalf("syncParentDirectory() error = %v, want nil", err)
	}
}
