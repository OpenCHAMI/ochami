// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build windows

package configfile

// Windows does not expose a portable directory fsync operation. Rename still
// provides atomic replacement; file data was flushed before the rename.
func syncParentDirectory(string) error { return nil }
