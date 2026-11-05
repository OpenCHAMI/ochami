// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build linux && integration

package widget

import "testing"

// TestWidget_LinuxIntegration verifies widgets against a live service on Linux.
// The integration tag doesn't let a file hold a subject's tests apart from the
// other Linux tests in widget_linux_test.go.
func TestWidget_LinuxIntegration(t *testing.T) { t.Error("split from its platform's file") }
