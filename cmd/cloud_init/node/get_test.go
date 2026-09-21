// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package node

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestNodeGetHeaderOptionsAreCommandLocal(t *testing.T) {
	t.Parallel()

	commands := []struct {
		name string
		new  func() *cobra.Command
	}{
		{name: "group", new: newCmdNodeGetGroup},
		{name: "user-data", new: newCmdNodeGetUserdata},
		{name: "vendor-data", new: newCmdNodeGetVendordata},
	}

	for _, tt := range commands {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			first := tt.new()
			second := tt.new()
			if err := first.Flags().Set("headers", "never"); err != nil {
				t.Fatalf("setting first command headers: %v", err)
			}
			if got := second.Flags().Lookup("headers").Value.String(); got != "multiple" {
				t.Errorf("second command headers = %q, want %q", got, "multiple")
			}
		})
	}
}
