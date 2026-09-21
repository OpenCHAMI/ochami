// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build windows

package config

// syncParentDirectory does nothing on Windows, which has no portable directory
// fsync. The rename still replaces the file atomically, and the file data was
// flushed before the rename.
func syncParentDirectory(string) error { return nil }
