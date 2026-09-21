// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"strings"
	"testing"
)

// TestMetacommandPaths_PrintUsage verifies every metacommand path (a command
// with no direct action of its own) prints usage and exits successfully when
// invoked without a subcommand.
func TestMetacommandPaths_PrintUsage(t *testing.T) {
	paths := [][]string{
		{},
		{"boot"}, {"boot", "bmc"}, {"boot", "config"}, {"boot", "node"}, {"boot", "service"},
		{"bss"}, {"bss", "boot"}, {"bss", "boot", "image"}, {"bss", "boot", "params"},
		{"bss", "boot", "script"}, {"bss", "hosts"}, {"bss", "service"},
		{"cloud-init"}, {"cloud-init", "defaults"}, {"cloud-init", "group"}, {"cloud-init", "group", "get"}, {"cloud-init", "node"}, {"cloud-init", "node", "get"}, {"cloud-init", "service"},
		{"config"}, {"config", "cluster"}, {"discover"},
		{"metadata"}, {"metadata", "defaults"}, {"metadata", "group"}, {"metadata", "instance"}, {"metadata", "peer"}, {"metadata", "service"},
		{"pcs"}, {"pcs", "service"}, {"pcs", "status"}, {"pcs", "transition"},
		{"rcs"}, {"rcs", "console"}, {"rcs", "service"},
		{"smd"}, {"smd", "compep"}, {"smd", "component"}, {"smd", "group"}, {"smd", "group", "member"},
		{"smd", "iface"}, {"smd", "rfe"}, {"smd", "service"},
	}

	for _, path := range paths {
		name := "root"
		if len(path) > 0 {
			name = strings.Join(path, " ")
		}
		t.Run(name, func(t *testing.T) {
			args := append(append([]string{}, path...), "--ignore-config")
			res := runOchamiWithRuntime(t, args...)
			if res.err != nil {
				t.Fatalf("unexpected error: %v", res.err)
			}
			if !strings.Contains(res.stdout, "Usage:") {
				t.Errorf("stdout = %q, want usage", res.stdout)
			}
		})
	}
}
