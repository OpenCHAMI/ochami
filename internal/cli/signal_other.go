// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build !unix

package cli

import "os"

// dieFromSignal does nothing: outside Unix, a process can't end itself by a
// signal for its parent to observe, so the caller exits with the signal's
// code instead.
func dieFromSignal(os.Signal) {}
