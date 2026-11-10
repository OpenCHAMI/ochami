// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package widget

import "testing"

// TestGadget_Rejects verifies Set rejects a bad gadget, in a file that doesn't
// share a stem with the subject's plain tests in gadget_test.go.
func TestGadget_Rejects(t *testing.T) { t.Error("in a file with another stem") }
