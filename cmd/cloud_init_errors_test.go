// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"strings"
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

// TestCloudInitServiceStatus_NotRunning verifies that when the service is
// unreachable, "cloud-init service status" reports not running and resolves to
// CodeNetwork.
func TestCloudInitServiceStatus_NotRunning(t *testing.T) {
	url := "http://127.0.0.1:1" // nothing listens on port 1, so connections are refused

	res := runOchami(t, "cloud-init", "service", "status", "--ignore-config", "--uri", url)

	if res.err == nil {
		t.Fatal("expected an error, got nil")
	}
	if res.exitCode != cli.CodeNetwork {
		t.Errorf("exit code = %d, want %d (%s)", res.exitCode, cli.CodeNetwork, cli.CodeName(cli.CodeNetwork))
	}
	if !strings.Contains(res.stdout, "cloud-init is not running") {
		t.Errorf("stdout = %q, want it to report not running", res.stdout)
	}
}
