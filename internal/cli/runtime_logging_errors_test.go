// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// TestInitLogging_RejectsInvalidConfiguration verifies that InitLogging rejects
// an invalid configured log level.
func TestInitLogging_RejectsInvalidConfiguration(t *testing.T) {
	rt := NewTestRuntime(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	rt.Config.Log.Level = "not-a-level"
	rt.Config.Log.Format = "json"
	rt.Config.Log.Color = "off"
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("log-format", "", "")
	cmd.Flags().String("log-level", "", "")
	cmd.Flags().String("log-color", "", "")
	if err := rt.InitLogging(cmd); err == nil {
		t.Fatal("InitLogging() accepted invalid level")
	}
}
