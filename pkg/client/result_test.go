// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package client

import (
	"errors"
	"reflect"
	"testing"
)

// TestBatchResultHelpers verifies successful values and item errors are filtered in input order.
func TestBatchResultHelpers(t *testing.T) {
	wantErr := errors.New("failed")
	results := BatchResult[int]{
		{Value: 1},
		{Value: 2, Err: wantErr},
		{Value: 3},
	}

	if !results.HasErrors() {
		t.Fatal("HasErrors() = false, want true")
	}
	if got := results.Errors(); !reflect.DeepEqual(got, []error{wantErr}) {
		t.Errorf("Errors() = %v, want [%v]", got, wantErr)
	}
	if got := results.Values(); !reflect.DeepEqual(got, []int{1, 3}) {
		t.Errorf("Values() = %v, want [1 3]", got)
	}
	if (BatchResult[int]{}).HasErrors() {
		t.Fatal("empty HasErrors() = true, want false")
	}
}
