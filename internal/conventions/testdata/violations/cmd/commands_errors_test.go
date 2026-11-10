// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import "testing"

// TestServiceCommands_NetworkError verifies every service command fails when
// its service can't be reached. Error tables across command families are kept
// in this file, whatever their plain tests' files are named.
func TestServiceCommands_NetworkError(t *testing.T) { t.Error("allowed another stem") }
