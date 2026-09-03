// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package cloud_init

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestCIFlagHeaderWhen(t *testing.T) {
	var value CIFlagHeaderWhen
	for _, input := range []string{CIFlagHeaderAlways, CIFlagHeaderMultiple, CIFlagHeaderNever} {
		if err := value.Set(input); err != nil {
			t.Errorf("Set(%q): %v", input, err)
		}
	}
	if err := value.Set("invalid"); err == nil {
		t.Error("Set(invalid) returned nil")
	}
	if value.Type() != "CIFlagHeaderWhen" || value.String() != CIFlagHeaderNever {
		t.Errorf("value = %q, type = %q", value.String(), value.Type())
	}
	values, directive := CompletionHeaderWhen(&cobra.Command{}, nil, "")
	if len(values) != 3 || directive != cobra.ShellCompDirectiveDefault {
		t.Errorf("completion = %v, %v", values, directive)
	}
}
