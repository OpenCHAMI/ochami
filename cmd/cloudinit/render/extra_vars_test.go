// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package render

import (
	"errors"
	"io"
	"path/filepath"
	"testing"
)

type extraVarsErrorReader struct{ err error }

func (r extraVarsErrorReader) Read([]byte) (int, error) { return 0, r.err }

// TestReadExtraVars_SourceErrors verifies stdin errors, missing files, and nil
// stdin are reported, while inline input leaves the reader untouched.
func TestReadExtraVars_SourceErrors(t *testing.T) {
	t.Parallel()
	want := io.ErrUnexpectedEOF
	if _, err := readExtraVars("@-", extraVarsErrorReader{err: want}); !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
	if _, err := readExtraVars("@-", nil); err == nil {
		t.Fatal("expected an error for nil stdin")
	}
	if _, err := readExtraVars("@"+filepath.Join(t.TempDir(), "missing"), nil); err == nil {
		t.Fatal("expected a missing file error")
	}
	if data, err := readExtraVars("{}", extraVarsErrorReader{err: want}); err != nil || string(data) != "{}" {
		t.Fatalf("inline data = %q, error = %v, want {} without error", data, err)
	}
}
