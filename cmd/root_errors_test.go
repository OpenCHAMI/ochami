// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestRootCommand_RejectsCACertWithInsecure verifies that passing --cacert
// and --insecure together is a usage error. The two are contradictory:
// --insecure skips certificate verification entirely, while UseCACert always
// re-enables it once a CA is loaded, so allowing both would silently make
// --insecure a no-op whenever --cacert is also set.
func TestRootCommand_RejectsCACertWithInsecure(t *testing.T) {
	t.Parallel()

	res := runOchamiWithRuntime(t, "--ignore-config", "--insecure", "--cacert", "/path/to/ca.pem", "version")
	if res.err == nil {
		t.Fatal("passing --cacert and --insecure together = nil error, want usage error")
	}
	if res.exitCode != cli.CodeUsage {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeUsage, cli.CodeName(cli.CodeUsage))
	}
}
