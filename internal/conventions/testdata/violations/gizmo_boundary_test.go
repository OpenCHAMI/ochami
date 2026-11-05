// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build unix

package widget

import "testing"

// TestGadget_Unix verifies gadgets on Unix. A platform-specific file may have a
// different stem from the subject's other files.
func TestGadget_Unix(t *testing.T) { t.Error("allowed another stem") }
