// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

//go:build integration

package widget

import "testing"

// TestWidget_Integration verifies widgets against a live service. A build tag
// that doesn't name a platform doesn't let a file hold a subject's tests apart
// from its other files.
func TestWidget_Integration(t *testing.T) { t.Error("split from its subject's file") }
