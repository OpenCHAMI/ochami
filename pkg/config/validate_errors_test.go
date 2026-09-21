// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import "testing"

// TestValidateConfig_Errors verifies that validateConfig rejects a timeout that
// isn't a duration and a cluster enable-auth value that isn't a boolean.
func TestValidateConfig_Errors(t *testing.T) {
	// Invalid timeout duration.
	if err := validateConfig(loadKoanfYAML(t, "timeout: not-a-duration\n")); err == nil {
		t.Error("validateConfig(invalid timeout) = nil, want error")
	}

	// Invalid enable-auth value.
	yaml := `clusters:
- name: demo
  cluster:
    enable-auth: maybe
`
	if err := validateConfig(loadKoanfYAML(t, yaml)); err == nil {
		t.Error("validateConfig(invalid enable-auth) = nil, want error")
	}
}
