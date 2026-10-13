// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

// Package widget breaks every source convention on purpose; see
// TestChecks_FlagFixtures. The go tool ignores testdata, so this file is
// only parsed, never built.
package widget

import (
	"testing"

	"github.com/openchami/ochami/internal/cli"
)

func Test_widgetName(t *testing.T) {}

// TestWidget verifies a widget exists.
func TestWidget(t *testing.T) { t.Log("widget") }

// TestWidget_Error exercises the error arm, like TestWidget does.
func TestWidget_Error(t *testing.T) {
	if cli.ExitCode(nil) != 0 {
		t.Errorf("exit = %d, want %d", cli.ExitCode(nil), cli.CodeSuccess)
	}
}

// TestWidget_Get verifies Get issues GET /widgets/{id} and returns CodeHTTP.
func TestWidget_Get(t *testing.T) {
	t.Error("checks neither the method, the path, nor the code")
}
