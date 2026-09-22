// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"testing"
)

func TestNewKeyValPatchData_RejectsInvalidRemoveIndex(t *testing.T) {
	_, _, err := NewKeyValPatchData(nil, nil, nil, []string{"groups=-"})
	if err == nil {
		t.Fatalf("NewKeyValPatchData accepted invalid remove index")
	}
}
