// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build !unix

package config

import "os"

// fileOwner reports false: file ownership isn't a user and group ID pair
// outside Unix, so there is nothing for writeFile to preserve.
func fileOwner(os.FileInfo) (uid, gid int, ok bool) { return 0, 0, false }
