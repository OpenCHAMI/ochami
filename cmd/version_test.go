// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

// version_test.go verifies the "version" command prints build metadata and
// exits successfully.

import (
	"strings"
	"testing"
)

// TestVersion verifies "ochami version" exits successfully and prints the
// version metadata fields to stdout.
func TestVersion(t *testing.T) {
	res := runOchami(t, "version", "--ignore-config")
	if res.err != nil {
		t.Fatalf("unexpected error: %v (exit %d)", res.err, res.exitCode)
	}
	for _, field := range []string{"Version:", "Commit:", "Build Host:", "Build User:"} {
		if !strings.Contains(res.stdout, field) {
			t.Errorf("stdout = %q, want it to contain %q", res.stdout, field)
		}
	}
}
